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

package dns

import (
	"context"
	"sort"

	"emperror.dev/errors"
	"github.com/banzaicloud/operator-tools/pkg/reconciler"
	"github.com/banzaicloud/operator-tools/pkg/utils"
	"golang.org/x/mod/semver"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"

	"github.com/banzaicloud/pipeline/internal/common"
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

	resourceReconciler := reconciler.NewReconcilerWith(cli)
	_, object, err := resourceReconciler.CreateIfNotExist(&incomingSI, reconciler.StateCreated)
	if err != nil {
		return errors.Wrap(err, "failed to create the service instance resource")
	}

	existingSI, ok := object.(*v1alpha1.ServiceInstance)
	if !ok {
		return errors.Wrap(err, "failed to create the service instance resource")
	}

	// at this point the incoming changes need to be applied to the existing instance - that'll be updated
	existingSI.Spec.Enabled = incomingSI.Spec.Enabled
	// make sure the flag is populated / enable it by default
	if existingSI.Spec.Enabled == nil {
		existingSI.Spec.Enabled = utils.BoolPointer(true)
	}

	existingSI.Spec.Version = incomingSI.Spec.Version
	// make sure the version is populated / set the latest available version by default
	if incomingSI.Spec.Version == "" {
		latestVersion, err := is.getLatestVersion(*existingSI)
		if err != nil {
			return errors.Wrap(err, "failed to get the  latest version")
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

	key, okErr := client.ObjectKeyFromObject(&incomingSI)
	if okErr != nil {
		return errors.Wrap(err, "failed to get object key for lookup")
	}

	existingSI := v1alpha1.ServiceInstance{}
	if err := cli.Get(ctx, key, &existingSI); err != nil {
		if errors2.IsNotFound(err) {
			// resource is not found
			return nil
		}
		return errors.Wrap(err, "failed to look up service instance")
	}

	existingSI.Spec.Enabled = utils.BoolPointer(false) // effectively disable the service instance
	if _, err := reconciler.NewReconcilerWith(cli).ReconcileResource(&existingSI, reconciler.StatePresent); err != nil {
		return errors.Wrap(err, "failed to reconcile the integrated service")
	}

	return nil
}

func (is isvcReconciler) getLatestVersion(instance v1alpha1.ServiceInstance) (string, error) {
	if len(instance.Status.AvailableVersions) == 0 {
		return "", errors.New("no versions available")
	}

	availableVersions := make(AvailableVersions, 0, len(instance.Status.AvailableVersions))
	for version := range instance.Status.AvailableVersions {
		availableVersions = append(availableVersions, version)
	}

	// sort the available versions
	sort.Sort(availableVersions)

	// get the highest version available
	return availableVersions[len(availableVersions)-1], nil
}

// AvailableVersions slice of semver version strings to facilitate sorting
type AvailableVersions []string

func (a AvailableVersions) Len() int {
	return len(a)
}

func (a AvailableVersions) Less(i, j int) bool {
	if semver.Compare(a[i], a[j]) < 0 {
		return true
	}

	return false
}

func (a AvailableVersions) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
