// Copyright © 2021 Banzai Cloud
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
	"github.com/banzaicloud/operator-tools/pkg/utils"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type integratedServiceCleaner struct {
	scheme       *runtime.Scheme
	kubeConfigFn ClusterKubeConfigFunc
}

func NewIntegratedServiceClean(kubeConfigFn ClusterKubeConfigFunc) IntegratedServiceCleaner {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)

	return integratedServiceCleaner{
		scheme:       scheme,
		kubeConfigFn: kubeConfigFn,
	}
}

func (r integratedServiceCleaner) DisableServiceInstance(ctx context.Context, clusterID uint) error {
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

	siClient := serviceInstanceClient{
		clusterClient: clusterClient,
		items:         lookupISvcs.Items,
	}

	// disable all service instance
	if err := siClient.disable(ctx); err != nil {
		return errors.WrapIf(err, "failed to disable integrated service(s)")
	}

	// wait for all service instance status will be uninstalled
	if err := siClient.wait(ctx); err != nil {
		return errors.WrapIf(err, "failed to wait for integrated service(s) uninstalled")
	}

	return nil
}

func (r integratedServiceCleaner) k8sClientForCluster(ctx context.Context, clusterID uint) (client.Client, error) {
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

type serviceInstanceClient struct {
	clusterClient client.Client
	items         []v1alpha1.ServiceInstance
}

func (c serviceInstanceClient) check(ctx context.Context, key types.NamespacedName) error {
	// wait till the status becomes uninstalled or uninstallFailed
	for {
		inactiveSI := v1alpha1.ServiceInstance{}
		if err := c.clusterClient.Get(ctx, key, &inactiveSI); err != nil {
			if apiErrors.IsNotFound(err) {
				// resource is not found
				return nil
			}
			return errors.Wrap(err, "failed to look up service instance")
		}

		if inactiveSI.Status.Phase == v1alpha1.UninstallFailed {
			return errors.New("failed to uninstall integrated service")
		}

		if inactiveSI.Status.Phase == v1alpha1.Uninstalled {
			return nil
		}

		// sleep a bit for the reconcile to proceed
		time.Sleep(2 * time.Second)
	}
}

func (c serviceInstanceClient) disable(ctx context.Context) error {
	var err error
	for _, item := range c.items {
		if utils.PointerToBool(item.Spec.Enabled) {
			item.Spec.Enabled = utils.BoolPointer(false)
			err = errors.Append(err, c.clusterClient.Update(ctx, &item))
		}
	}

	return err
}

func (c serviceInstanceClient) wait(ctx context.Context) error {
	var err error
	for _, item := range c.items {
		key := types.NamespacedName{
			Namespace: item.Namespace,
			Name:      item.Name,
		}

		err = errors.Append(err, c.check(ctx, key))
	}

	return err
}
