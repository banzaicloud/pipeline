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

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/secret"
)

func TestFeatureManager_Name(t *testing.T) {
	mng := MakeFeatureManager(nil, nil, nil, nil, NewFeatureConfiguration(), nil)

	assert.Equal(t, "monitoring", mng.Name())
}

func TestFeatureManager_GetOutput(t *testing.T) {
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

	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	helmService := dummyHelmService{}
	endpointService := dummyEndpointService{}
	logger := commonadapter.NewNoopLogger()
	config := NewFeatureConfiguration()
	mng := MakeFeatureManager(clusterGetter, secretStore, endpointService, helmService, config, logger)
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

	assert.Equal(t, clusterfeature.FeatureOutput{
		"grafana": obj{
			"serviceUrl": serviceUrl,
			"url":        grafanaURL,
		},
		"prometheus": obj{
			"serviceUrl": serviceUrl,
			"url":        prometheusURL,
		},
		"prometheusOperator": obj{
			"version": config.operator.chartVersion,
		},
		"alertmanager": obj{
			"serviceUrl": serviceUrl,
		},
		"pushgateway": obj{},
	}, output)
}

func TestFeatureManager_ValidateSpec(t *testing.T) {
	config := NewFeatureConfiguration()
	mng := MakeFeatureManager(nil, nil, nil, nil, config, nil)

	cases := map[string]struct {
		Spec  clusterfeature.FeatureSpec
		Error interface{}
	}{
		"empty spec": {
			Spec:  clusterfeature.FeatureSpec{},
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
					"enabled":          true,
					"nodeExporter":     true,
					"kubeStateMetrics": true,
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
					"enabled":          true,
					"nodeExporter":     true,
					"kubeStateMetrics": true,
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
					"enabled":          true,
					"nodeExporter":     true,
					"kubeStateMetrics": true,
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
					"enabled":          true,
					"nodeExporter":     false,
					"kubeStateMetrics": true,
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
					"enabled":          true,
					"nodeExporter":     true,
					"kubeStateMetrics": false,
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
				assert.True(t, clusterfeature.IsInputValidationError(err))
			case false, nil:
				assert.NoError(t, err)
			default:
				assert.Equal(t, tc.Error, errors.Cause(err))
			}
		})
	}
}
