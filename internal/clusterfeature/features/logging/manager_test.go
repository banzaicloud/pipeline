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

package logging

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
	mng := MakeFeatureManager(nil, nil, nil, Config{}, nil)

	assert.Equal(t, "logging", mng.Name())
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
				lokiSecretID: {
					ID:      lokiSecretID,
					Name:    getLokiSecretName(clusterID),
					Type:    secrettype.HtpasswdSecretType,
					Values:  map[string]string{secrettype.Username: "admin", secrettype.Password: "pass"},
					Tags:    []string{secret.TagBanzaiReadonly},
					Version: 1,
				},
			},
		},
	}

	config := Config{
		Charts: ChartsConfig{
			Operator: ChartConfig{
				Version: "1.0.0",
			},
		},
		Images: ImagesConfig{
			Operator: ImageConfig{
				Tag: "v2.0.0",
			},
			Loki: ImageConfig{
				Tag: "v2.0.0",
			},
			Fluentbit: ImageConfig{
				Tag: "v3.0.1",
			},
			Fluentd: ImageConfig{
				Tag: "v3.0.2",
			},
		},
	}

	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	endpointService := dummyEndpointService{}
	logger := commonadapter.NewNoopLogger()
	mng := MakeFeatureManager(clusterGetter, secretStore, endpointService, config, logger)
	ctx := auth.SetCurrentOrganizationID(context.Background(), orgID)

	spec := obj{
		"loki": obj{
			"enabled": true,
			"ingress": obj{
				"enabled": true,
				"path":    "/loki",
			},
		},
		"logging": obj{
			"metrics": true,
			"tls":     true,
		},
		"clusterOutput": obj{
			"enabled": false,
		},
	}

	output, err := mng.GetOutput(ctx, clusterID, spec)
	assert.NoError(t, err)

	assert.Equal(t, clusterfeature.FeatureOutput{
		"logging": obj{
			"operatorVersion":  "1.0.0",
			"fluentdVersion":   "v3.0.2",
			"fluentbitVersion": "v3.0.1",
		},
		"loki": obj{
			"secretId":   "",
			"version":    "v2.0.0",
			"url":        lokiURL,
			"serviceUrl": lokiServiceUrl,
		},
	}, output)
}

func TestFeatureManager_ValidateSpec(t *testing.T) {
	mng := MakeFeatureManager(nil, nil, nil, Config{}, nil)

	cases := map[string]struct {
		Spec  clusterfeature.FeatureSpec
		Error interface{}
	}{
		"empty spec": {
			Spec:  clusterfeature.FeatureSpec{},
			Error: false,
		},
		"valid spec": {
			Spec: clusterfeature.FeatureSpec{
				"loki": obj{
					"enabled": true,
					"ingress": obj{
						"enabled": true,
						"path":    "/loki",
					},
				},
				"logging": obj{
					"metrics": true,
					"tls":     true,
				},
				"clusterOutput": obj{
					"enabled": true,
					"provider": obj{
						"name":     "s3",
						"secretId": "asdasd",
						"bucket": obj{
							"name": "testbucket",
						},
					},
				},
			},
			Error: false,
		},
		"required bucket secret": {
			Spec: clusterfeature.FeatureSpec{
				"loki": obj{
					"enabled": true,
					"ingress": obj{
						"enabled": true,
						"path":    "/loki",
					},
				},
				"logging": obj{
					"metrics": true,
					"tls":     true,
				},
				"clusterOutput": obj{
					"enabled": true,
					"provider": obj{
						"name":     "oss",
						"secretId": "",
						"bucket": obj{
							"name": "testbucket",
						},
					},
				},
			},
			Error: true,
		},
		"storageaccount required": {
			Spec: clusterfeature.FeatureSpec{
				"loki": obj{
					"enabled": true,
					"ingress": obj{
						"enabled": true,
						"path":    "/loki",
					},
				},
				"logging": obj{
					"metrics": true,
					"tls":     true,
				},
				"clusterOutput": obj{
					"enabled": true,
					"provider": obj{
						"name":     "azure",
						"secretId": "asdasd",
						"bucket": obj{
							"name":          "testbucket",
							"resourceGroup": "testrg",
						},
					},
				},
			},
			Error: true,
		},
		"resourcegroup required": {
			Spec: clusterfeature.FeatureSpec{
				"loki": obj{
					"enabled": true,
					"ingress": obj{
						"enabled": true,
						"path":    "/loki",
					},
				},
				"logging": obj{
					"metrics": true,
					"tls":     true,
				},
				"clusterOutput": obj{
					"enabled": true,
					"provider": obj{
						"name":     "azure",
						"secretId": "asdasd",
						"bucket": obj{
							"name":           "testbucket",
							"storageAccount": "testsa",
						},
					},
				},
			},
			Error: true,
		},
		"invalid bucket provider": {
			Spec: clusterfeature.FeatureSpec{
				"loki": obj{
					"enabled": true,
					"ingress": obj{
						"enabled": true,
						"path":    "/loki",
					},
				},
				"logging": obj{
					"metrics": true,
					"tls":     true,
				},
				"clusterOutput": obj{
					"enabled": true,
					"provider": obj{
						"name":     "amazon",
						"secretId": "asdasd",
						"bucket": obj{
							"name": "testbucket",
						},
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
				assert.True(t, clusterfeature.IsInputValidationError(err))
			case false, nil:
				assert.NoError(t, err)
			default:
				assert.Equal(t, tc.Error, errors.Cause(err))
			}
		})
	}
}
