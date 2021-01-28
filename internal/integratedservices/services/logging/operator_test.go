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

	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/secret"
)

func TestIntegratedServiceOperator_Name(t *testing.T) {
	op := MakeIntegratedServicesOperator(nil, nil, nil, nil, nil, Config{}, nil, nil)

	assert.Equal(t, "logging", op.Name())
}

func TestIntegratedServiceOperator_Apply(t *testing.T) {
	clusterID := uint(42)
	orgID := uint(13)

	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]dummyCluster{},
	}
	clusterService := integratedserviceadapter.NewClusterService(clusterGetter)
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
			},
		},
	}

	logger := services.NoopLogger{}
	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	kubernetesService := dummyKubernetesService{}
	endpointService := dummyEndpointService{}
	op := MakeIntegratedServicesOperator(clusterGetter, clusterService, helmService, &kubernetesService, endpointService, Config{}, logger, secretStore)

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
		"enable Loki": {
			Spec: integratedservices.IntegratedServiceSpec{
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

func TestIntegratedServiceOperator_Deactivate(t *testing.T) {
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
	clusterService := integratedserviceadapter.NewClusterService(clusterGetter)
	helmService := dummyHelmService{}
	endpointService := dummyEndpointService{}
	orgSecretStore := dummyOrganizationalSecretStore{
		Secrets: map[uint]map[string]*secret.SecretItemResponse{
			orgID: nil,
		},
	}
	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	logger := services.NoopLogger{}
	kubernetesService := dummyKubernetesService{}
	op := MakeIntegratedServicesOperator(clusterGetter, clusterService, helmService, &kubernetesService, endpointService, Config{}, logger, secretStore)

	ctx := context.Background()

	_ = op.Deactivate(ctx, clusterID, nil)
}
