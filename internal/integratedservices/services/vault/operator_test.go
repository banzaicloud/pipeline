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

package vault

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/secret"
)

func TestIntegratedServiceOperator_Name(t *testing.T) {
	op := MakeIntegratedServicesOperator(nil, nil, nil, nil, nil, Config{}, nil)

	assert.Equal(t, "vault", op.Name())
}

func testIntegratedServiceOperatorApply(t *testing.T) {
	clusterID := uint(42)
	orgID := uint(13)

	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]dummyCluster{},
	}
	clusterService := integratedserviceadapter.NewClusterService(clusterGetter)
	helmService := dummyHelmService{}
	kubernetesService := dummyKubernetesService{}

	orgSecretStore := dummyOrganizationalSecretStore{
		Secrets: map[uint]map[string]*secret.SecretItemResponse{
			orgID: nil,
		},
	}

	logger := services.NoopLogger{}
	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	op := MakeIntegratedServicesOperator(clusterGetter, clusterService, helmService, &kubernetesService, secretStore, Config{}, logger)

	cases := map[string]struct {
		Spec    integratedservices.IntegratedServiceSpec
		Cluster dummyCluster
		Error   interface{}
	}{
		"cluster not ready": {
			Spec: integratedservices.IntegratedServiceSpec{},
			Cluster: dummyCluster{
				OrgID:  orgID,
				Status: pkgCluster.Creating,
				ID:     clusterID,
			},
			Error: integratedservices.ClusterIsNotReadyError{
				ClusterID: clusterID,
			},
		},
		"Pipeline's Vault": {
			Spec: integratedservices.IntegratedServiceSpec{
				"customVault": obj{
					"enabled": false,
				},
				"settings": obj{
					"namespaces":      []string{"default"},
					"serviceAccounts": []string{"*"},
				},
			},
			Cluster: dummyCluster{
				OrgID:  orgID,
				Status: pkgCluster.Running,
				ID:     clusterID,
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

func TestIntegratedServiceOperator_Deactivate(t *testing.T) {
	clusterID := uint(42)

	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]dummyCluster{
			clusterID: {
				Status: pkgCluster.Running,
				ID:     clusterID,
			},
		},
	}
	clusterService := integratedserviceadapter.NewClusterService(clusterGetter)
	helmService := dummyHelmService{}
	kubernetesService := dummyKubernetesService{}
	logger := services.NoopLogger{}
	op := MakeIntegratedServicesOperator(clusterGetter, clusterService, helmService, &kubernetesService, nil, Config{}, logger)

	ctx := context.Background()

	_ = op.Deactivate(ctx, clusterID, nil)
}
