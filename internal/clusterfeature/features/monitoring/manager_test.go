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
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
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
		Clusters: map[uint]clusterfeatureadapter.Cluster{
			clusterID: dummyCluster{
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
					Type:    pkgSecret.Password,
					Values:  map[string]string{pkgSecret.Username: "admin", pkgSecret.Password: "pass"},
					Tags:    []string{pkgSecret.TagBanzaiReadonly},
					Version: 1,
				},
				prometheusSecretID: {
					ID:      prometheusSecretID,
					Name:    getPrometheusSecretName(clusterID),
					Type:    pkgSecret.Password,
					Values:  map[string]string{pkgSecret.Username: "admin", pkgSecret.Password: "pass"},
					Tags:    []string{pkgSecret.TagBanzaiReadonly},
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
			"public": obj{
				"enabled": true,
				"path":    "/grafana",
			},
			"secretId": grafanaSecretID,
		},
		"alertmanager": obj{
			"enabled": true,
			"public": obj{
				"enabled": false,
			},
		},
		"prometheus": obj{
			"enabled": true,
			"public": obj{
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
			"url": grafanaURL,
		},
		"prometheus": obj{
			"url": prometheusURL,
		},
		"prometheusOperator": obj{
			"version": config.operator.chartVersion,
		},
		"alertmanager": obj{},
		"pushgateway":  obj{},
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
			Error: false,
		},
		"valid spec": {
			Spec: obj{
				"grafana": obj{
					"enabled": true,
					"public": obj{
						"enabled": true,
						"path":    grafanaPath,
					},
				},
				"prometheus": obj{
					"enabled": true,
					"public": obj{
						"enabled": true,
						"path":    prometheusPath,
					},
				},
			},
			Error: false,
		},
		"Grafana path empty": {
			Spec: obj{
				"grafana": obj{
					"enabled": true,
					"public": obj{
						"enabled": true,
						"path":    "",
					},
				},
			},
			Error: true,
		},
		"invalid domain": {
			Spec: obj{
				"grafana": obj{
					"enabled": true,
					"public": obj{
						"enabled": true,
						"path":    grafanaPath,
						"domain":  "23445@#",
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
