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

type StubbedSecretMapper struct {
	mappedSpec  integratedservices.IntegratedServiceSpec
	mappedError error
}

func (f StubbedSecretMapper) MapSpec(ctx context.Context, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceSpec, error) {
	return f.mappedSpec, f.mappedError
}

func TestSpecTransformation(t *testing.T) {
	testData := map[string]struct {
		service      string
		mappers      map[string]integratedservices.SpecMapper
		serviceSpec  string
		managed      bool
		expectError  bool
		expectResult interface{}
	}{
		"managed service mapped successfully": {
			service:     "fake-is",
			serviceSpec: `{}`,
			mappers: map[string]integratedservices.SpecMapper{
				"fake-is": StubbedSecretMapper{
					mappedSpec: map[string]interface{}{
						"key": "val",
					},
					mappedError: nil,
				},
			},
			managed:     true,
			expectError: false,
			expectResult: map[string]interface{}{
				"key": "val",
			},
		},
		"managed service fails to map": {
			service:     "fake-is",
			serviceSpec: `{}`,
			mappers: map[string]integratedservices.SpecMapper{
				"fake-is": StubbedSecretMapper{
					mappedSpec:  map[string]interface{}{},
					mappedError: errors.NewPlain("cannot be mapped"),
				},
			},
			managed:      true,
			expectError:  true,
			expectResult: nil,
		},
		"unmanaged service does not get mapped": {
			service:     "fake-is",
			serviceSpec: `{"a": "b"}`,
			mappers: map[string]integratedservices.SpecMapper{
				"fake-is": StubbedSecretMapper{
					mappedSpec: map[string]interface{}{
						"c": "d",
					},
					mappedError: nil,
				},
			},
			managed:     false,
			expectError: false,
			expectResult: map[string]interface{}{
				"a": "b",
			},
		},
		"unmanaged service not affected by mapping": {
			service:     "fake-is",
			serviceSpec: `{"a": "b"}`,
			mappers: map[string]integratedservices.SpecMapper{
				"fake-is": StubbedSecretMapper{
					mappedSpec: map[string]interface{}{
						"c": "d",
					},
					mappedError: errors.NewPlain("does not propagate"),
				},
			},
			managed:     false,
			expectError: false,
			expectResult: map[string]interface{}{
				"a": "b",
			},
		},
	}

	for name, d := range testData {
		t.Log(name)
		specTransformation := NewSpecTransformation(services.NewServiceStatusMapper(), d.mappers)
		serviceInstance := v1alpha1.ServiceInstance{
			ObjectMeta: v1.ObjectMeta{
				Name: d.service,
			},
			Spec: v1alpha1.ServiceInstanceSpec{
				ServiceSpec: d.serviceSpec,
			},
		}

		if d.managed {
			services.SetManagedByPipeline(&serviceInstance.ObjectMeta)
		}

		transformedSpec, err := specTransformation.Transform(context.TODO(), serviceInstance)
		if d.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, d.expectResult, transformedSpec.Spec)
		}
	}
}
