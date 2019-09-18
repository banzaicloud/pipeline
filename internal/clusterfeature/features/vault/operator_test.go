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

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
)

func TestFeatureOperator_Name(t *testing.T) {
	op := MakeFeatureOperator(nil, nil, nil, nil, nil, nil)

	assert.Equal(t, "vault", op.Name())
}

func TestFeatureOperator_Apply(t *testing.T) {
	clusterID := uint(42)
	orgID := uint(13)

	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]clusterfeatureadapter.Cluster{},
	}
	clusterService := clusterfeatureadapter.NewClusterService(clusterGetter)
	helmService := dummyHelmService{}
	kubernetesService := dummyKubernetesService{}

	orgSecretStore := dummyOrganizationalSecretStore{
		Secrets: map[uint]map[string]*secret.SecretItemResponse{
			orgID: nil,
		},
	}

	logger := commonadapter.NewNoopLogger()
	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	op := MakeFeatureOperator(clusterGetter, clusterService, helmService, &kubernetesService, secretStore, logger)

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
			},
			Error: clusterfeature.ClusterIsNotReadyError{
				ClusterID: clusterID,
			},
		},
		"Pipeline's Vault": {
			Spec: clusterfeature.FeatureSpec{
				"customVault": obj{
					"enabled": false,
				},
				"settings": obj{
					"namespaces":      []string{"default"},
					"serviceAccounts": []string{"*"},
				},
			},
			Cluster: dummyCluster{
				OrgID: orgID,
				Ready: true,
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

	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]clusterfeatureadapter.Cluster{
			clusterID: dummyCluster{
				Ready: true,
			},
		},
	}
	clusterService := clusterfeatureadapter.NewClusterService(clusterGetter)
	helmService := dummyHelmService{}
	kubernetesService := dummyKubernetesService{}
	logger := commonadapter.NewNoopLogger()
	op := MakeFeatureOperator(clusterGetter, clusterService, helmService, &kubernetesService, nil, logger)

	ctx := context.Background()

	_ = op.Deactivate(ctx, clusterID, nil)
}
