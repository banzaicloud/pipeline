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
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
)

type specConversionStub struct {
	mappedSpec  integratedservices.IntegratedServiceSpec
	mappedError error
}

func (f specConversionStub) ConvertSpec(ctx context.Context, service v1alpha1.ServiceInstance) (integratedservices.IntegratedServiceSpec, error) {
	return f.mappedSpec, f.mappedError
}

func TestServiceConversion(t *testing.T) {
	testData := map[string]struct {
		service      string
		mappers      map[string]integratedservices.SpecConversion
		serviceSpec  string
		expectError  bool
		expectResult interface{}
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
			expectError: false,
			expectResult: map[string]interface{}{
				"key": "val",
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
			expectError:  true,
			expectResult: nil,
		},
		"service not recognized": {
			service:      "fake-is",
			mappers:      map[string]integratedservices.SpecConversion{},
			expectError:  true,
			expectResult: nil,
		},
	}

	for name, d := range testData {
		t.Log(name)
		serviceTransformation := NewServiceConversion(services.NewServiceStatusMapper(), d.mappers)
		serviceInstance := v1alpha1.ServiceInstance{
			ObjectMeta: v1.ObjectMeta{
				Name: d.service,
			},
			Spec: v1alpha1.ServiceInstanceSpec{
				Service: d.service,
			},
		}

		transformedService, err := serviceTransformation.Convert(context.TODO(), serviceInstance)
		if d.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, d.expectResult, transformedService.Spec)
		}
	}
}
