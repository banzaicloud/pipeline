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
	"testing"

	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
)

type specConversionStub struct {
	mappedSpec  integratedservices.IntegratedServiceSpec
	mappedError error
}

func (f specConversionStub) ConvertSpec(_ context.Context, _ v1alpha1.ServiceInstance) (integratedservices.IntegratedServiceSpec, error) {
	return f.mappedSpec, f.mappedError
}

type outputResolverStub struct {
	resolvedOutput integratedservices.IntegratedServiceOutput
	resolvedError  error
}

func (o outputResolverStub) Resolve(_ context.Context, _ v1alpha1.ServiceInstance) (integratedservices.IntegratedServiceOutput, error) {
	return o.resolvedOutput, o.resolvedError
}

func TestServiceConversion(t *testing.T) {
	testData := map[string]struct {
		service         string
		mappers         map[string]integratedservices.SpecConversion
		outputResolvers map[string]OutputResolver
		serviceSpec     string
		expect          func(t *testing.T, i integratedservices.IntegratedService, err error)
	}{
		"spec extracted successfully": {
			service: "fake-is",
			mappers: map[string]integratedservices.SpecConversion{
				"fake-is": specConversionStub{
					mappedSpec: map[string]interface{}{
						"key": "val",
					},
					mappedError: nil,
				},
			},
			expect: func(t *testing.T, i integratedservices.IntegratedService, err error) {
				require.NoError(t, err)
				require.Equal(t, map[string]interface{}{
					"key": "val",
				}, i.Spec)
			},
		},
		"spec fails to map": {
			service: "fake-is",
			mappers: map[string]integratedservices.SpecConversion{
				"fake-is": specConversionStub{
					mappedSpec:  map[string]interface{}{},
					mappedError: errors.NewPlain("cannot be mapped"),
				},
			},
			expect: func(t *testing.T, i integratedservices.IntegratedService, err error) {
				require.Error(t, err)
			},
		},
		"service not recognized": {
			service: "fake-is",
			mappers: map[string]integratedservices.SpecConversion{},
			expect: func(t *testing.T, i integratedservices.IntegratedService, err error) {
				require.Error(t, err)
			},
		},
		"custom output": {
			service: "fake-is",
			mappers: map[string]integratedservices.SpecConversion{
				"fake-is": specConversionStub{},
			},
			outputResolvers: map[string]OutputResolver{
				"fake-is": outputResolverStub{
					resolvedOutput: map[string]interface{}{
						"a": "b",
					},
					resolvedError: nil,
				},
			},
			expect: func(t *testing.T, i integratedservices.IntegratedService, err error) {
				require.NoError(t, err)
				require.Equal(t, map[string]interface{}{
					"a": "b",
				}, i.Output)
			},
		},
		"failed output": {
			service: "fake-is",
			mappers: map[string]integratedservices.SpecConversion{
				"fake-is": specConversionStub{},
			},
			outputResolvers: map[string]OutputResolver{
				"fake-is": outputResolverStub{
					resolvedOutput: map[string]interface{}{
						"a": "b",
					},
					resolvedError: errors.NewPlain("asd"),
				},
			},
			expect: func(t *testing.T, i integratedservices.IntegratedService, err error) {
				require.Error(t, err)
			},
		},
	}

	for name, d := range testData {
		t.Run(name, func(t *testing.T) {
			serviceTransformation := NewServiceConversion(services.NewServiceStatusMapper(), d.mappers, d.outputResolvers)
			serviceInstance := v1alpha1.ServiceInstance{
				ObjectMeta: v1.ObjectMeta{
					Name: d.service,
				},
				Spec: v1alpha1.ServiceInstanceSpec{
					Service: d.service,
				},
			}

			transformedService, err := serviceTransformation.Convert(context.TODO(), serviceInstance)
			d.expect(t, transformedService, err)
		})
	}
}
