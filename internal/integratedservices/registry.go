// Copyright © 2019 Banzai Cloud
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

package integratedservices

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	"github.com/banzaicloud/operator-tools/pkg/reconciler"
	"github.com/banzaicloud/operator-tools/pkg/utils"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MakeIntegratedServiceManagerRegistry returns a IntegratedServiceManagerRegistry with the specified integrated service managers registered.
func MakeIntegratedServiceManagerRegistry(managers []IntegratedServiceManager) IntegratedServiceManagerRegistry {
	lookup := make(map[string]IntegratedServiceManager, len(managers))
	for _, fm := range managers {
		lookup[fm.Name()] = fm
	}

	return integratedServiceManagerRegistry{
		lookup: lookup,
	}
}

type integratedServiceManagerRegistry struct {
	lookup map[string]IntegratedServiceManager
}

func (r integratedServiceManagerRegistry) GetIntegratedServiceManager(integratedServiceName string) (IntegratedServiceManager, error) {
	if integratedServiceManager, ok := r.lookup[integratedServiceName]; ok {
		return integratedServiceManager, nil
	}

	return nil, errors.WithStack(UnknownIntegratedServiceError{IntegratedServiceName: integratedServiceName})
}

func (r integratedServiceManagerRegistry) GetIntegratedServiceNames() []string {
	keys := make([]string, 0)
	for key := range r.lookup {
		keys = append(keys, key)
	}
	return keys
}

// MakeIntegratedServiceOperatorRegistry returns a IntegratedServiceOperatorRegistry with the specified integrated service operators registered.
func MakeIntegratedServiceOperatorRegistry(operators []IntegratedServiceOperator, kubeConfigFn ClusterKubeConfigFunc) IntegratedServiceOperatorRegistry {
	lookup := make(map[string]IntegratedServiceOperator, len(operators))
	for _, fo := range operators {
		lookup[fo.Name()] = fo
	}

	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)

	return integratedServiceOperatorRegistry{
		lookup:       lookup,
		scheme:       scheme,
		kubeConfigFn: kubeConfigFn,
	}
}

type integratedServiceOperatorRegistry struct {
	lookup map[string]IntegratedServiceOperator

	scheme       *runtime.Scheme
	kubeConfigFn ClusterKubeConfigFunc
}

func (r integratedServiceOperatorRegistry) GetIntegratedServiceOperator(integratedServiceName string) (IntegratedServiceOperator, error) {
	if integratedServiceOperator, ok := r.lookup[integratedServiceName]; ok {
		return integratedServiceOperator, nil
	}

	return nil, errors.WithStack(UnknownIntegratedServiceError{IntegratedServiceName: integratedServiceName})
}

func (r integratedServiceOperatorRegistry) DisableServiceInstance(ctx context.Context, clusterID uint) error {
	clusterClient, err := r.k8sClientForCluster(ctx, clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to build cluster client")
	}

	lookupISvcs := &v1alpha1.ServiceInstanceList{}
	if err := clusterClient.List(ctx, lookupISvcs); err != nil {
		if meta.IsNoMatchError(err) {
			return nil
		}
		return errors.Wrap(err, "failed to retrieve service instance list")
	}

	kubeConfig, err := r.kubeConfigFn.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get K8S config")
	}

	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create rest config from cluster configuration")
	}

	cli, err := client.New(restCfg, client.Options{Scheme: r.scheme})
	if err != nil {
		return errors.Wrap(err, "failed to create the client from rest configuration")
	}

	for _, item := range lookupISvcs.Items {

		if item.ObjectMeta.Annotations["app.kubernetes.io/managed-by"] == "banzaicloud.io/pipeline" {

			item.Spec.Enabled = utils.BoolPointer(false)

			if _, err := reconciler.NewReconcilerWith(cli).ReconcileResource(&item, reconciler.StatePresent); err != nil {
				return errors.Wrap(err, "failed to reconcile the integrated service")
			}

			incomingSI := v1alpha1.ServiceInstance{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: item.Namespace,
					Name:      item.Spec.Service,
				},
			}

			key, err := client.ObjectKeyFromObject(&incomingSI)
			if err != nil {
				return errors.Wrap(err, "failed to get object key for lookup")
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

			if _, err := reconciler.NewReconcilerWith(cli).ReconcileResource(&item, reconciler.StatePresent); err != nil {
				return errors.Wrap(err, "failed to reconcile the integrated service")
			}
		}
	}

	return nil
}

func (r integratedServiceOperatorRegistry) k8sClientForCluster(ctx context.Context, clusterID uint) (client.Client, error) {
	kubeConfig, err := r.kubeConfigFn.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve the k8s config")
	}

	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create rest config from cluster configuration")
	}

	cli, err := client.New(restCfg, client.Options{Scheme: r.scheme})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create the client from rest configuration")
	}

	return cli, nil
}

// UnknownIntegratedServiceError is returned when there is no integrated service manager registered for a integrated service.
type UnknownIntegratedServiceError struct {
	IntegratedServiceName string
}

func (UnknownIntegratedServiceError) Error() string {
	return "unknown integrated service"
}

// Details returns the error's details
func (e UnknownIntegratedServiceError) Details() []interface{} {
	return []interface{}{"integratedService", e.IntegratedServiceName}
}

// ServiceError tells the transport layer whether this error should be translated into the transport format
// or an internal error should be returned instead.
func (UnknownIntegratedServiceError) ServiceError() bool {
	return true
}

// Unknown tells a client that this error is related to a resource being unsupported.
// Can be used to translate the error to eg. status code.
func (UnknownIntegratedServiceError) Unknown() bool {
	return true
}
