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

package monitoring

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/secret"
)

func TestIntegratedServiceManager_Name(t *testing.T) {
	mng := MakeIntegratedServiceManager(nil, nil, nil, nil, Config{}, nil)

	assert.Equal(t, "monitoring", mng.Name())
}

func TestIntegratedServiceManager_GetOutput(t *testing.T) {
	orgID := uint(13)
	clusterID := uint(42)
	clusterName := "the-cluster"

	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]dummyCluster{
			clusterID: {
				Name:  clusterName,
				OrgID: orgID,
				ID:    clusterID,
			},
		},
	}

	orgSecretStore := dummyOrganizationalSecretStore{
		Secrets: map[uint]map[string]*secret.SecretItemResponse{
			orgID: {
				grafanaSecretID: {
					ID:      grafanaSecretID,
					Name:    getGrafanaSecretName(clusterID),
					Type:    secrettype.Password,
					Values:  map[string]string{secrettype.Username: "admin", secrettype.Password: "pass"},
					Tags:    []string{secret.TagBanzaiReadonly},
					Version: 1,
				},
				prometheusSecretID: {
					ID:      prometheusSecretID,
					Name:    getPrometheusSecretName(clusterID),
					Type:    secrettype.Password,
					Values:  map[string]string{secrettype.Username: "admin", secrettype.Password: "pass"},
					Tags:    []string{secret.TagBanzaiReadonly},
					Version: 1,
				},
			},
		},
	}

	config := Config{
		Grafana: GrafanaConfig{AdminUser: "admin"},
		Charts: ChartsConfig{
			Operator: ChartConfig{
				Version: "1.0.0",
			},
		},
		Images: ImagesConfig{
			Operator: ImageConfig{
				Tag: "v0.1.1",
			},
			Prometheus: ImageConfig{
				Tag: "v0.1.2",
			},
			Alertmanager: ImageConfig{
				Tag: "v0.1.3",
			},
			Grafana: ImageConfig{
				Tag: "v0.1.4",
			},
			Kubestatemetrics: ImageConfig{
				Tag: "v0.1.5",
			},
			Nodeexporter: ImageConfig{
				Tag: "v0.1.6",
			},
			Pushgateway: ImageConfig{
				Tag: "v0.1.7",
			},
		},
	}

	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	helmService := dummyHelmService{}
	endpointService := dummyEndpointService{}
	logger := services.NoopLogger{}
	mng := MakeIntegratedServiceManager(clusterGetter, secretStore, endpointService, helmService, config, logger)
	ctx := auth.SetCurrentOrganizationID(context.Background(), orgID)

	spec := obj{
		"grafana": obj{
			"enabled": true,
			"ingress": obj{
				"enabled": true,
				"path":    "/grafana",
			},
			"secretId": grafanaSecretID,
		},
		"alertmanager": obj{
			"enabled": true,
			"ingress": obj{
				"enabled": false,
			},
		},
		"prometheus": obj{
			"enabled": true,
			"ingress": obj{
				"enabled": true,
				"path":    "/prometheus",
			},
			"secretId": prometheusSecretID,
		},
	}

	output, err := mng.GetOutput(ctx, clusterID, spec)
	assert.NoError(t, err)

	assert.Equal(t, integratedservices.IntegratedServiceOutput{
		"grafana": obj{
			"serviceUrl": serviceUrl,
			"url":        grafanaURL,
			"version":    "v0.1.4",
		},
		"prometheus": obj{
			"serviceUrl": serviceUrl,
			"url":        prometheusURL,
			"version":    "v0.1.2",
		},
		"prometheusOperator": obj{
			"version": config.Charts.Operator.Version,
		},
		"alertmanager": obj{
			"serviceUrl": serviceUrl,
			"version":    "v0.1.3",
		},
		"pushgateway": obj{
			"version": "v0.1.7",
		},
	}, output)
}

func TestIntegratedServiceManager_ValidateSpec(t *testing.T) {
	mng := MakeIntegratedServiceManager(nil, nil, nil, nil, Config{}, nil)

	cases := map[string]struct {
		Spec  integratedservices.IntegratedServiceSpec
		Error interface{}
	}{
		"empty spec": {
			Spec:  integratedservices.IntegratedServiceSpec{},
			Error: true,
		},
		"valid spec": {
			Spec: obj{
				"grafana": obj{
					"enabled": true,
					"ingress": obj{
						"enabled": true,
						"path":    grafanaPath,
					},
				},
				"prometheus": obj{
					"enabled": true,
					"storage": obj{
						"size":      100,
						"retention": "10m",
					},
					"ingress": obj{
						"enabled": true,
						"path":    prometheusPath,
					},
				},
				"exporters": obj{
					"enabled": true,
					"nodeExporter": obj{
						"enabled": true,
					},
					"kubeStateMetrics": obj{
						"enabled": true,
					},
				},
			},
			Error: false,
		},
		"Grafana path empty": {
			Spec: obj{
				"grafana": obj{
					"enabled": true,
					"ingress": obj{
						"enabled": true,
						"path":    "",
					},
				},
				"prometheus": obj{
					"enabled": true,
					"storage": obj{
						"size":      100,
						"retention": "10m",
					},
					"ingress": obj{
						"enabled": true,
						"path":    prometheusPath,
					},
				},
				"exporters": obj{
					"enabled": true,
					"nodeExporter": obj{
						"enabled": true,
					},
					"kubeStateMetrics": obj{
						"enabled": true,
					},
				},
			},
			Error: true,
		},
		"Grafana invalid domain": {
			Spec: obj{
				"grafana": obj{
					"enabled": true,
					"ingress": obj{
						"enabled": true,
						"domain":  "2342#@",
						"path":    grafanaPath,
					},
				},
				"prometheus": obj{
					"enabled": true,
					"storage": obj{
						"size":      100,
						"retention": "10m",
					},
					"ingress": obj{
						"enabled": true,
						"path":    prometheusPath,
					},
				},
				"exporters": obj{
					"enabled": true,
					"nodeExporter": obj{
						"enabled": true,
					},
					"kubeStateMetrics": obj{
						"enabled": true,
					},
				},
			},
			Error: true,
		},
		"disabled exporters": {
			Spec: obj{
				"grafana": obj{
					"enabled": true,
					"ingress": obj{
						"enabled": true,
						"path":    grafanaPath,
					},
				},
				"prometheus": obj{
					"enabled": true,
					"storage": obj{
						"size":      100,
						"retention": "10m",
					},
					"ingress": obj{
						"enabled": true,
						"path":    prometheusPath,
					},
				},
				"exporters": obj{
					"enabled": false,
				},
			},
			Error: true,
		},
		"disabled nodeExporter": {
			Spec: obj{
				"grafana": obj{
					"enabled": true,
					"ingress": obj{
						"enabled": true,
						"path":    grafanaPath,
					},
				},
				"prometheus": obj{
					"enabled": true,
					"storage": obj{
						"size":      100,
						"retention": "10m",
					},
					"ingress": obj{
						"enabled": true,
						"path":    prometheusPath,
					},
				},
				"exporters": obj{
					"enabled": true,
					"nodeExporter": obj{
						"enabled": false,
					},
					"kubeStateMetrics": obj{
						"enabled": true,
					},
				},
			},
			Error: true,
		},
		"disabled kubeStateMetrics": {
			Spec: obj{
				"grafana": obj{
					"enabled": true,
					"ingress": obj{
						"enabled": true,
						"path":    grafanaPath,
					},
				},
				"prometheus": obj{
					"enabled": true,
					"storage": obj{
						"size":      100,
						"retention": "10m",
					},
					"ingress": obj{
						"enabled": true,
						"path":    prometheusPath,
					},
				},
				"exporters": obj{
					"enabled": true,
					"nodeExporter": obj{
						"enabled": true,
					},
					"kubeStateMetrics": obj{
						"enabled": false,
					},
				},
			},
			Error: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			err := mng.ValidateSpec(ctx, tc.Spec)
			switch tc.Error {
			case true:
				assert.True(t, integratedservices.IsInputValidationError(err))
			case false, nil:
				assert.NoError(t, err)
			default:
				assert.Equal(t, tc.Error, errors.Cause(err))
			}
		})
	}
}
