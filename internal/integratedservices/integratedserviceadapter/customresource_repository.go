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
	"encoding/json"

	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
)

// crRepository repository implementation that directly accesses a custom resources in a kubernetes cluster
type crRepository struct {
	scheme             *runtime.Scheme
	kubeConfigFn       integratedservices.ClusterKubeConfigFunc
	specTransformation *SpecTransformation
	logger             common.Logger
	namespace          string
}

// Creates a new CR repository instance to access integrated services in a k8s cluster
func NewCRRepository(kubeConfigFn integratedservices.ClusterKubeConfigFunc, logger common.Logger, specTransformation *SpecTransformation, namespace string) integratedservices.IntegratedServiceRepository {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)

	return crRepository{
		scheme:             scheme,
		kubeConfigFn:       kubeConfigFn,
		namespace:          namespace,
		logger:             logger,
		specTransformation: specTransformation,
	}
}

func (c crRepository) GetIntegratedServices(ctx context.Context, clusterID uint) ([]integratedservices.IntegratedService, error) {
	clusterClient, err := c.k8sClientForCluster(ctx, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build cluster client")
	}

	lookupISvcs := &v1alpha1.ServiceInstanceList{}
	if err := clusterClient.List(ctx, lookupISvcs); err != nil {
		return nil, errors.Wrap(err, "failed to retrieve integrated service list")
	}

	iSvcs := make([]integratedservices.IntegratedService, 0, len(lookupISvcs.Items))
	for _, si := range lookupISvcs.Items {
		transformed, err := c.specTransformation.Transform(ctx, si)
		if err != nil {
			continue
		}
		iSvcs = append(iSvcs, transformed)
	}

	return iSvcs, nil
}

func (c crRepository) GetIntegratedService(ctx context.Context, clusterID uint, serviceName string) (integratedservices.IntegratedService, error) {
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
	key, err := client.ObjectKeyFromObject(&lookupSI)
	if err != nil {
		return emptyIS, errors.Wrap(err, "failed to get object key for lookup")
	}

	if err := clusterClient.Get(ctx, key, &lookupSI); err != nil {
		if apiErrors.IsNotFound(err) {
			return emptyIS, integratedServiceNotFoundError{
				ClusterID:             clusterID,
				IntegratedServiceName: serviceName,
			}
		}

		return emptyIS, errors.Wrap(err, "failed to look up service instance")
	}

	return c.specTransformation.Transform(ctx, lookupSI)
}

func (c crRepository) SaveIntegratedService(_ context.Context, _ uint, _ string, _ integratedservices.IntegratedServiceSpec, _ string) error {
	// NO op
	return nil
}

func (c crRepository) UpdateIntegratedServiceStatus(_ context.Context, _ uint, _ string, _ string) error {
	// NO op
	return nil
}

func (c crRepository) UpdateIntegratedServiceSpec(_ context.Context, _ uint, _ string, _ integratedservices.IntegratedServiceSpec) error {
	// NO op
	return nil
}

func (c crRepository) DeleteIntegratedService(_ context.Context, _ uint, _ string) error {
	// NO op
	return nil
}

// k8sClientForCluster builds a client that accesses the cluster
// TODO the built client should be a caching one? (revise this)
func (c crRepository) k8sClientForCluster(ctx context.Context, clusterID uint) (client.Client, error) {
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

type SpecTransformation struct {
	statusMapper services.StatusMapper
	specMappers  map[string]integratedservices.SpecMapper
}

func NewSpecTransformation(statusMapper services.StatusMapper, specMappers map[string]integratedservices.SpecMapper) *SpecTransformation {
	return &SpecTransformation{
		statusMapper: statusMapper,
		specMappers:  specMappers,
	}
}

func (c SpecTransformation) Transform(ctx context.Context, instance v1alpha1.ServiceInstance) (integratedservices.IntegratedService, error) {
	var serviceSpec integratedservices.IntegratedServiceSpec

	if instance.Spec.ServiceSpec != "" {
		if err := json.Unmarshal([]byte(instance.Spec.ServiceSpec), &serviceSpec); err != nil {
			return integratedservices.IntegratedService{}, errors.WrapIf(err, "failed to decode api spec")
		}
	}

	if services.IsManagedByPipeline(instance.ObjectMeta) {
		for name, mapper := range c.specMappers {
			if name == instance.ObjectMeta.Name {
				mappedServiceSpec, err := mapper.MapSpec(ctx, serviceSpec)
				if err != nil {
					return integratedservices.IntegratedService{}, errors.WrapIfWithDetails(err,
						"failed to map service spec", "service", instance.Spec.Service)
				}
				serviceSpec = mappedServiceSpec
				break
			}
		}
	}

	return integratedservices.IntegratedService{
		Name:   instance.Name,
		Spec:   serviceSpec,
		Status: c.statusMapper.MapStatus(instance),
	}, nil
}
