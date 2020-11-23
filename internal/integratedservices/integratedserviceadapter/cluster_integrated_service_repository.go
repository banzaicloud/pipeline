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

	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

// clusterRepository repository implementation that directly accesses a cluster for resource operations
type clusterRepository struct {
	scheme       *runtime.Scheme
	kubeConfigFn integratedservices.ClusterKubeConfigFunc
	specWrapper  integratedservices.SpecWrapper
	namespace    string
}

// Creates a new cluster repository to access Integrated services in a k8s cluster
func NewClusterRepository(kubeConfigFn integratedservices.ClusterKubeConfigFunc, wrapper integratedservices.SpecWrapper) integratedservices.IntegratedServiceRepository {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)

	return clusterRepository{
		scheme:       scheme,
		kubeConfigFn: kubeConfigFn,
		specWrapper:  wrapper,
		namespace:    "default", // TODO make the namespace dynamic (integrated services ara always deployed to pipeline-system ?!)
	}
}

func (c clusterRepository) GetIntegratedServices(ctx context.Context, clusterID uint) ([]integratedservices.IntegratedService, error) {
	client, err := c.k8sClientForCluster(ctx, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build cluster client")
	}

	lookupISvcs := &v1alpha1.ServiceInstanceList{}
	if err := client.List(ctx, lookupISvcs); err != nil {
		return nil, errors.Wrap(err, "failed to retrieve integrated service list")
	}

	iSvcs := make([]integratedservices.IntegratedService, 0, len(lookupISvcs.Items))
	for _, si := range lookupISvcs.Items {
		transformed, err := c.transform(si)
		if err != nil {
			continue
		}
		iSvcs = append(iSvcs, transformed)
	}

	return iSvcs, nil
}

func (c clusterRepository) GetIntegratedService(ctx context.Context, clusterID uint, serviceName string) (integratedservices.IntegratedService, error) {
	emptyIS := integratedservices.IntegratedService{}
	clusterClient, err := c.k8sClientForCluster(ctx, clusterID)
	if err != nil {
		return emptyIS, errors.Wrap(err, "failed to build cluster client")
	}

	lookupSI := v1alpha1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: c.namespace,
		},
	}
	key, okErr := client.ObjectKeyFromObject(&lookupSI)
	if okErr != nil {
		return emptyIS, errors.Wrap(err, "failed to get object key for lookup")
	}

	if err := clusterClient.Get(ctx, key, &lookupSI); err != nil {
		if errors2.IsNotFound(err) {
			return emptyIS, integratedServiceNotFoundError{
				ClusterID:             clusterID,
				IntegratedServiceName: serviceName,
			}
		}

		return emptyIS, errors.Wrap(err, "failed to look up service instance")
	}

	return c.transform(lookupSI)
}

func (c clusterRepository) SaveIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string, spec integratedservices.IntegratedServiceSpec, status string) error {
	// NO op
	return nil
}

func (c clusterRepository) UpdateIntegratedServiceStatus(ctx context.Context, clusterID uint, integratedServiceName string, status string) error {
	// NO op
	return nil
}

func (c clusterRepository) UpdateIntegratedServiceSpec(ctx context.Context, clusterID uint, integratedServiceName string, spec integratedservices.IntegratedServiceSpec) error {
	// NO op
	return nil
}

func (c clusterRepository) DeleteIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string) error {
	// NO op
	return nil
}

// k8sClientForCluster builds a client that accesses the cluster
// TODO the built client should be a caching one? (revise this)
func (c clusterRepository) k8sClientForCluster(ctx context.Context, clusterID uint) (client.Client, error) {
	kubeConfig, err := c.kubeConfigFn.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve the k8s config")
	}

	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create rest config from cluster configuration")
	}

	cli, err := client.New(restCfg, client.Options{Scheme: c.scheme})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create the client from rest configuration")
	}

	return cli, nil
}

func (c clusterRepository) transform(instance v1alpha1.ServiceInstance) (integratedservices.IntegratedService, error) {
	transformedIS := integratedservices.IntegratedService{}
	transformedIS.Name = instance.Name

	apiSpec, err := c.specWrapper.Unwrap(context.Background(), []byte(instance.Spec.Config))
	if err != nil {
		return integratedservices.IntegratedService{}, errors.Wrap(err, "failed to unwarap configuration from custom resource")
	}

	transformedIS.Spec = apiSpec
	transformedIS.Status = c.getISStatus(instance)

	return transformedIS, nil
}

func (c clusterRepository) getISStatus(instance v1alpha1.ServiceInstance) integratedservices.IntegratedServiceStatus {
	// TODO refine the status mapping
	switch instance.Status.Phase {
	case v1alpha1.Installed:
		return integratedservices.IntegratedServiceStatusActive
	case v1alpha1.InstallFailed:
		return integratedservices.IntegratedServiceStatusError
	case "":
		return integratedservices.IntegratedServiceStatusInactive
	}

	return integratedservices.IntegratedServiceStatusPending
}
