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

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/secret"
)

func TestFeatureOperator_Name(t *testing.T) {
	op := MakeFeatureOperator(nil, nil, nil, nil, nil, Config{}, nil, nil)

	assert.Equal(t, "logging", op.Name())
}

func TestFeatureOperator_Apply(t *testing.T) {
	clusterID := uint(42)
	orgID := uint(13)

	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]dummyCluster{},
	}
	clusterService := clusterfeatureadapter.NewClusterService(clusterGetter)
	helmService := dummyHelmService{}

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
				alibabaSecretID: {
					ID:      alibabaSecretID,
					Name:    alibabaSecretName,
					Type:    secrettype.Alibaba,
					Values:  map[string]string{secrettype.AlibabaAccessKeyId: "asd", secrettype.AlibabaSecretAccessKey: "asd"},
					Tags:    []string{secret.TagBanzaiReadonly},
					Version: 1,
				},
			},
		},
	}

	logger := commonadapter.NewNoopLogger()
	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	kubernetesService := dummyKubernetesService{}
	endpointService := dummyEndpointService{}
	op := MakeFeatureOperator(clusterGetter, clusterService, helmService, &kubernetesService, endpointService, Config{}, logger, secretStore)

	cases := map[string]struct {
		Spec    clusterfeature.FeatureSpec
		Cluster dummyCluster
		Error   interface{}
	}{
		"cluster not ready": {
			Spec: clusterfeature.FeatureSpec{},
			Cluster: dummyCluster{
				OrgID:  orgID,
				Status: pkgCluster.Creating,
				ID:     clusterID,
			},
			Error: clusterfeature.ClusterIsNotReadyError{
				ClusterID: clusterID,
			},
		},
		"enable Loki": {
			Spec: clusterfeature.FeatureSpec{
				"loki": obj{
					"enabled": true,
					"ingress": obj{
						"enabled": false,
					},
				},
				"logging": obj{
					"metrics": true,
					"tls":     false,
				},
				"clusterOutput": obj{
					"enabled": false,
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

func TestFeatureOperator_Deactivate(t *testing.T) {
	clusterID := uint(42)
	orgID := uint(13)

	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]dummyCluster{
			clusterID: {
				Status: pkgCluster.Running,
				ID:     clusterID,
			},
		},
	}
	clusterService := clusterfeatureadapter.NewClusterService(clusterGetter)
	helmService := dummyHelmService{}
	endpointService := dummyEndpointService{}
	orgSecretStore := dummyOrganizationalSecretStore{
		Secrets: map[uint]map[string]*secret.SecretItemResponse{
			orgID: nil,
		},
	}
	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	logger := commonadapter.NewNoopLogger()
	kubernetesService := dummyKubernetesService{}
	op := MakeFeatureOperator(clusterGetter, clusterService, helmService, &kubernetesService, endpointService, Config{}, logger, secretStore)

	ctx := context.Background()

	_ = op.Deactivate(ctx, clusterID, nil)
}
