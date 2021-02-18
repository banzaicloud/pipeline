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

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
)

type ServiceConversion struct {
	statusMapper    services.StatusMapper
	specConversion  map[string]integratedservices.SpecConversion
	outputResolvers map[string]OutputResolver
}

type OutputResolver interface {
	Resolve(ctx context.Context, instance v1alpha1.ServiceInstance) (integratedservices.IntegratedServiceOutput, error)
}

func NewServiceConversion(
	statusMapper services.StatusMapper,
	specConverters map[string]integratedservices.SpecConversion,
	outputResolvers map[string]OutputResolver) *ServiceConversion {
	return &ServiceConversion{
		statusMapper:    statusMapper,
		specConversion:  specConverters,
		outputResolvers: outputResolvers,
	}
}

func (c ServiceConversion) Convert(ctx context.Context, instance v1alpha1.ServiceInstance) (integratedservices.IntegratedService, error) {
	if conversion, ok := c.specConversion[instance.Spec.Service]; ok {
		mappedServiceSpec, err := conversion.ConvertSpec(ctx, instance)
		if err != nil {
			return integratedservices.IntegratedService{}, errors.WrapIfWithDetails(err,
				"failed to convert service spec", "service", instance.Spec.Service)
		}
		convertedService := integratedservices.IntegratedService{
			Name:   instance.Name,
			Spec:   mappedServiceSpec,
			Status: c.statusMapper.MapStatus(instance),
		}
		if outputResolver, ok := c.outputResolvers[instance.Spec.Service]; ok {
			output, err := outputResolver.Resolve(ctx, instance)
			if err != nil {
				return integratedservices.IntegratedService{}, errors.WrapIfWithDetails(err, "output resolution failed for service",
					"name", instance.Name, "service", instance.Spec.Service)
			}

			// signal whether the resource is managed by pipeline in the output
			if services.IsManagedByPipeline(instance.ObjectMeta) {
				output["managed-by"] = "pipeline"
			}

			convertedService.Output = output
		}
		return convertedService, nil
	}
	return integratedservices.IntegratedService{}, errors.NewWithDetails("spec converter not found for service",
		"name", instance.Name, "service", instance.Spec.Service)
}
