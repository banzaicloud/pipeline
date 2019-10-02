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

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/stretchr/testify/assert"
)

func TestFeatureOperator_Name(t *testing.T) {
	op := MakeFeatureOperator(nil, nil, nil, NewFeatureConfiguration(), nil, nil)

	assert.Equal(t, "monitoring", op.Name())
}

func TestFeatureOperator_Apply(t *testing.T) {
	clusterID := uint(42)
	orgID := uint(13)

	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]clusterfeatureadapter.Cluster{},
	}
	clusterService := clusterfeatureadapter.NewClusterService(clusterGetter)
	helmService := dummyHelmService{}

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

	logger := commonadapter.NewNoopLogger()
	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	op := MakeFeatureOperator(clusterGetter, clusterService, helmService, Configuration{}, logger, secretStore)

	cases := map[string]struct {
		Spec    clusterfeature.FeatureSpec
		Cluster clusterfeatureadapter.Cluster
		Error   interface{}
	}{
		"cluster not ready": {
			Spec: clusterfeature.FeatureSpec{},
			Cluster: dummyCluster{
				OrgID: orgID,
				Ready: false,
				ID:    clusterID,
			},
			Error: clusterfeature.ClusterIsNotReadyError{
				ClusterID: clusterID,
			},
		},
		"Enabled Grafana and Alertmanager": {
			Spec: clusterfeature.FeatureSpec{
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
						"enabled": true,
						"path":    "/alertmanager",
					},
				},
			},
			Cluster: dummyCluster{
				OrgID: orgID,
				Ready: true,
				ID:    clusterID,
			},
			Error: false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			clusterGetter.Clusters[clusterID] = tc.Cluster

			ctx := auth.SetCurrentOrganizationID(context.Background(), orgID)

			err := op.Apply(ctx, clusterID, tc.Spec)
			switch tc.Error {
			case nil, false:
				assert.NoError(t, err)
			case true:
				assert.Error(t, err)
			default:
				assert.Equal(t, tc.Error, err)
			}
		})
	}
}

func TestFeatureOperator_Deactivate(t *testing.T) {
	clusterID := uint(42)
	orgID := uint(13)

	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]clusterfeatureadapter.Cluster{
			clusterID: dummyCluster{
				Ready: true,
				ID:    clusterID,
			},
		},
	}
	clusterService := clusterfeatureadapter.NewClusterService(clusterGetter)
	helmService := dummyHelmService{}
	orgSecretStore := dummyOrganizationalSecretStore{
		Secrets: map[uint]map[string]*secret.SecretItemResponse{
			orgID: nil,
		},
	}
	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	logger := commonadapter.NewNoopLogger()
	op := MakeFeatureOperator(clusterGetter, clusterService, helmService, Configuration{}, logger, secretStore)

	ctx := context.Background()

	_ = op.Deactivate(ctx, clusterID, nil)
}
