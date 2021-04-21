// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integratedserviceadapter

import (
	"context"
	"sort"
	"time"

	"emperror.dev/errors"
	"github.com/Masterminds/semver/v3"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	"github.com/banzaicloud/operator-tools/pkg/reconciler"
	"github.com/banzaicloud/operator-tools/pkg/utils"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
)

// Reconciler decouples handling of custom resources on a kubernetes cluster
type Reconciler interface {
	// Reconcile creates and applies CRs to a cluster
	Reconcile(ctx context.Context, kubeConfig []byte, svcInstance v1alpha1.ServiceInstance) error

	Disable(ctx context.Context, kubeConfig []byte, svcInstance v1alpha1.ServiceInstance) error
}

// isvcReconciler components struct in charge for assembling the CR manifest  and applying it to a cluster (by delegating to a cluster client)
type isvcReconciler struct {
	scheme *runtime.Scheme
	logger common.Logger
}

// NewISReconciler builds an integrated service reconciler
func NewISReconciler(logger common.Logger) Reconciler {
	// register needed shemes
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	return isvcReconciler{
		scheme: scheme,
		logger: logger,
	}
}

func (is isvcReconciler) Reconcile(ctx context.Context, kubeConfig []byte, incomingSI v1alpha1.ServiceInstance) error {
	is.logger.Debug("reconciling integrated service instance ...")

	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create rest config from cluster configuration")
	}

	cli, err := client.New(restCfg, client.Options{Scheme: is.scheme})
	if err != nil {
		return errors.Wrap(err, "failed to create the client from rest configuration")
	}

	services.SetManagedByPipeline(&incomingSI.ObjectMeta)

	resourceReconciler := reconciler.NewReconcilerWith(cli)
	isNew, object, err := resourceReconciler.CreateIfNotExist(&incomingSI, reconciler.StateCreated)
	if err != nil {
		return errors.Wrap(err, "failed to create the service instance resource")
	}

	existingSI := &v1alpha1.ServiceInstance{}
	if isNew {
		// retrieve the resource for the status data
		key, err := client.ObjectKeyFromObject(&incomingSI)
		if err != nil {
			return errors.Wrap(err, "failed to get object key for lookup")
		}

		// wait (endlessly) for the status of the newly created resource
		// in the edge case the status never gets populated, the routine wil be ended by the cadence worker!
		for {
			is.logger.Debug("Waiting for the service instance status ...")
			if err := cli.Get(ctx, key, existingSI); err != nil {
				return errors.Wrap(err, "failed to look up service instance")
			}

			// TODO use a specific error to signal shouldRetry
			if existingSI != nil && len(existingSI.Status.AvailableVersions) > 0 {
				is.logger.Debug("Service instance status populated.")
				// step forward
				break
			}

			// sleep a bit for the reconcile to proceed
			time.Sleep(2 * time.Second)
		}
	} else {
		var ok bool
		existingSI, ok = object.(*v1alpha1.ServiceInstance)
		if !ok {
			return errors.Errorf("service instance object conversion error %+v", object)
		}
	}

	if !services.IsManagedByPipeline(existingSI.ObjectMeta) {
		return errors.New("service instance is managed externally")
	}

	// at this point the incoming changes need to be applied to the existing instance - that'll be updated
	existingSI.Spec.Enabled = incomingSI.Spec.Enabled
	// make sure the flag is populated / enable it by default
	if existingSI.Spec.Enabled == nil {
		existingSI.Spec.Enabled = utils.BoolPointer(true)
	}

	existingSI.Spec.Version = incomingSI.Spec.Version
	existingSI.Spec.DNS = incomingSI.Spec.DNS
	existingSI.Spec.Backup = incomingSI.Spec.Backup
	// make sure the version is populated / set the latest available version by default
	if incomingSI.Spec.Version == "" {
		latestVersion, err := getLatestVersion(*existingSI)
		if err != nil {
			return errors.Wrap(err, "failed to get the latest version")
		}
		existingSI.Spec.Version = latestVersion
	}

	if _, err := resourceReconciler.ReconcileResource(existingSI, reconciler.StatePresent); err != nil {
		return errors.Wrap(err, "failed to reconcile the integrated service")
	}

	return nil
}

func (is isvcReconciler) Disable(ctx context.Context, kubeConfig []byte, incomingSI v1alpha1.ServiceInstance) error {
	is.logger.Debug("deactivating integrated service instance ...")

	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create rest config from cluster configuration")
	}

	cli, err := client.New(restCfg, client.Options{Scheme: is.scheme})
	if err != nil {
		return errors.Wrap(err, "failed to create the client from rest configuration")
	}

	key, err := client.ObjectKeyFromObject(&incomingSI)
	if err != nil {
		return errors.Wrap(err, "failed to get object key for lookup")
	}

	existingSI := v1alpha1.ServiceInstance{}
	if err := cli.Get(ctx, key, &existingSI); err != nil {
		if apiErrors.IsNotFound(err) {
			// resource is not found
			return nil
		}
		return errors.Wrap(err, "failed to look up service instance")
	}

	if !services.IsManagedByPipeline(existingSI.ObjectMeta) {
		return errors.New("service instance is managed externally")
	}

	existingSI.Spec.Enabled = utils.BoolPointer(false) // effectively disable the service instance
	if _, err := reconciler.NewReconcilerWith(cli).ReconcileResource(&existingSI, reconciler.StatePresent); err != nil {
		return errors.Wrap(err, "failed to reconcile the integrated service")
	}

	// wait till the status becomes uninstalled or uninstallFailed
	for {
		inactiveSI := v1alpha1.ServiceInstance{}
		if err := cli.Get(ctx, key, &inactiveSI); err != nil {
			if apiErrors.IsNotFound(err) {
				// resource is not found
				return nil
			}
			return errors.Wrap(err, "failed to look up service instance")
		}

		if inactiveSI.Status.Phase == v1alpha1.UninstallFailed {
			return errors.Wrap(err, "failed to uninstall integrated service")
		}

		if inactiveSI.Status.Phase == v1alpha1.Uninstalled {
			break
		}

		// sleep a bit for the reconcile to proceed
		time.Sleep(2 * time.Second)
	}

	if _, err := reconciler.NewReconcilerWith(cli).ReconcileResource(&existingSI, reconciler.StatePresent); err != nil {
		return errors.Wrap(err, "failed to reconcile the integrated service")
	}

	return nil
}

func getLatestVersion(instance v1alpha1.ServiceInstance) (string, error) {
	if len(instance.Status.AvailableVersions) == 0 {
		return "", errors.New("no versions available")
	}

	availableVersions := make([]*semver.Version, 0)

	var allErr error

	if instance.Status.Version != "" {
		actualVersion, err := semver.NewVersion(instance.Status.Version)
		if err != nil {
			allErr = errors.Combine(allErr, errors.Wrapf(err, "invalid version %s", instance.Status.Version))
		} else {
			availableVersions = append(availableVersions, actualVersion)
		}
	}

	for version := range instance.Status.AvailableVersions {
		parsedVersion, err := semver.NewVersion(version)
		if err != nil {
			allErr = errors.Combine(allErr, errors.Wrapf(err, "invalid version %s", version))
		} else {
			availableVersions = append(availableVersions, parsedVersion)
		}
	}

	if len(availableVersions) == 0 {
		return "", errors.WrapIf(allErr, "no valid versions available")
	}

	// sort the available versions
	sort.Sort(semver.Collection(availableVersions))

	// get the highest version available
	return availableVersions[len(availableVersions)-1].Original(), allErr
}
