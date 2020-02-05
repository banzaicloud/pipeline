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

package ingress

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
)

func TestSpec(t *testing.T) {
	type arr = []interface{}
	type obj = map[string]interface{}

	testCases := map[string]struct {
		Input      integratedservices.IntegratedServiceSpec
		Config     Config
		Expected   Spec
		Validation interface{}
	}{
		"traefik minimal": {
			Input: obj{
				"controller": obj{
					"type": "traefik",
				},
			},
			Config: Config{
				Controllers: []string{
					"traefik",
				},
			},
			Expected: Spec{
				Controller: ControllerSpec{
					Type: "traefik",
				},
			},
		},
		"traefik with default CN and SAN list": {
			Input: obj{
				"controller": obj{
					"type": "traefik",
					"config": obj{
						"ssl": obj{
							"defaultCN": "my.domain.org",
							"defaultSANList": arr{
								"my.domain.org",
								"*.my.domain.org",
							},
						},
					},
				},
			},
			Config: Config{
				Controllers: []string{
					"traefik",
				},
			},
			Expected: Spec{
				Controller: ControllerSpec{
					Type: "traefik",
					RawConfig: obj{
						"ssl": obj{
							"defaultCN": "my.domain.org",
							"defaultSANList": arr{
								"my.domain.org",
								"*.my.domain.org",
							},
						},
					},
				},
			},
		},
		"common parts": {
			Input: obj{
				"controller": obj{
					"type": "traefik",
				},
				"ingressClass": "my-ingress-class",
				"service": obj{
					"annotations": obj{
						"foo":  "bar",
						"fizz": "buzz",
					},
				},
			},
			Config: Config{
				Controllers: []string{
					"traefik",
				},
			},
			Expected: Spec{
				Controller: ControllerSpec{
					Type: "traefik",
				},
				IngressClass: "my-ingress-class",
				Service: ServiceSpec{
					Annotations: map[string]string{
						"foo":  "bar",
						"fizz": "buzz",
					},
				},
			},
		},
		"LoadBalancer service type": {
			Input: obj{
				"controller": obj{
					"type": "traefik",
				},
				"service": obj{
					"type": "LoadBalancer",
				},
			},
			Config: Config{
				Controllers: []string{
					"traefik",
				},
			},
			Expected: Spec{
				Controller: ControllerSpec{
					Type: "traefik",
				},
				Service: ServiceSpec{
					Type: "LoadBalancer",
				},
			},
		},
		"NodePort service type": {
			Input: obj{
				"controller": obj{
					"type": "traefik",
				},
				"service": obj{
					"type": "NodePort",
				},
			},
			Config: Config{
				Controllers: []string{
					"traefik",
				},
			},
			Expected: Spec{
				Controller: ControllerSpec{
					Type: "traefik",
				},
				Service: ServiceSpec{
					Type: "NodePort",
				},
			},
		},
		"ClusterIP service type": {
			Input: obj{
				"controller": obj{
					"type": "traefik",
				},
				"service": obj{
					"type": "ClusterIP",
				},
			},
			Config: Config{
				Controllers: []string{
					"traefik",
				},
			},
			Expected: Spec{
				Controller: ControllerSpec{
					Type: "traefik",
				},
				Service: ServiceSpec{
					Type: "ClusterIP",
				},
			},
		},
		"invalid service type": {
			Input: obj{
				"controller": obj{
					"type": "traefik",
				},
				"service": obj{
					"type": "NotAServiceType",
				},
			},
			Config: Config{
				Controllers: []string{
					"traefik",
				},
			},
			Expected: Spec{
				Controller: ControllerSpec{
					Type: "traefik",
				},
				Service: ServiceSpec{
					Type: "NotAServiceType",
				},
			},
			Validation: unsupportedServiceTypeError{
				ServiceType: "NotAServiceType",
			},
		},
		"unavailable controller type": {
			Input: obj{
				"controller": obj{
					"type": "traefik",
				},
			},
			Config: Config{
				Controllers: []string{
					"nginx",
					"istio",
				},
			},
			Expected: Spec{
				Controller: ControllerSpec{
					Type: "traefik",
				},
			},
			Validation: unavailableControllerError{
				Controller: "traefik",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			var spec Spec
			err := services.BindIntegratedServiceSpec(testCase.Input, &spec)
			require.NoError(t, err)
			require.Equal(t, testCase.Expected, spec)

			err = spec.Validate(testCase.Config)

			switch testCase.Validation {
			case nil, false:
				require.NoError(t, err)
			case true:
				require.Error(t, err)
			default:
				require.Equal(t, testCase.Validation, err)
			}
		})
	}
}
