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

package dns

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/pkg/brn"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/dns/route53"
)

func TestFeatureOperator_Name(t *testing.T) {
	op := MakeFeatureOperator(nil, nil, nil, nil, nil, nil, Config{})

	assert.Equal(t, "dns", op.Name())
}

func TestFeatureOperator_Apply(t *testing.T) {
	clusterID := uint(42)
	orgID := uint(13)
	providerSecretName := "route53-secret"
	providerSecretID := secret.GenerateSecretIDFromName(providerSecretName)

	orgSecretStore := dummyOrganizationalSecretStore{
		Secrets: map[uint]map[string]*secret.SecretItemResponse{
			orgID: {
				route53.IAMUserAccessKeySecretID: {
					ID:   route53.IAMUserAccessKeySecretID,
					Name: route53.IAMUserAccessKeySecretName,
					Type: pkgCluster.Amazon,
					Values: map[string]string{
						secrettype.AwsRegion:          "us-west-2",
						secrettype.AwsAccessKeyId:     "my-access-key-id",
						secrettype.AwsSecretAccessKey: "my-access-key-secret",
					},
				},
				providerSecretID: {
					ID:   providerSecretID,
					Name: providerSecretName,
					Type: pkgCluster.Amazon,
					Values: map[string]string{
						secrettype.AwsRegion:          "moon-21",
						secrettype.AwsAccessKeyId:     "an-access-key-id",
						secrettype.AwsSecretAccessKey: "an-access-key-secret",
					},
				},
			},
		},
	}
	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]dummyCluster{},
	}
	clusterService := clusterfeatureadapter.NewClusterService(clusterGetter)
	helmService := dummyHelmService{}
	logger := commonadapter.NewNoopLogger()
	orgDomainService := dummyOrgDomainService{
		Domain: "the.domain",
		OrgID:  orgID,
	}
	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	op := MakeFeatureOperator(clusterGetter, clusterService, helmService, logger, orgDomainService, secretStore, Config{})

	cases := map[string]struct {
		Spec    clusterfeature.FeatureSpec
		Cluster dummyCluster
		Error   interface{}
	}{
		"cluster ready": {
			Spec: clusterfeature.FeatureSpec{
				"clusterDomain": "cluster.org.the.domain",
				"externalDns": obj{
					"domainFilters": arr{
						"",
					},
					"provider": obj{
						"name":     "route53",
						"secretId": providerSecretID,
						"options": obj{
							"region":    "test-reg",
							"batchSize": 10,
						},
					},
					"txtOwnerId": "my-owner-id",
				},
			},
			Cluster: dummyCluster{
				OrgID:  orgID,
				Status: pkgCluster.Running,
			},
		},
		"cluster not ready": {
			Spec: clusterfeature.FeatureSpec{
				"clusterDomain": "cluster.org.the.domain",
				"externalDns": obj{
					"domainFilters": arr{
						"",
					},
					"provider": obj{
						"name":     "route53",
						"secretId": providerSecretID,
						"options": obj{
							"region":    "test-reg",
							"batchSize": 10,
						},
					},
					"txtOwnerId": "my-owner-id",
				},
			},
			Cluster: dummyCluster{
				OrgID:  orgID,
				Status: pkgCluster.Creating,
			},
			Error: clusterfeature.ClusterIsNotReadyError{
				ClusterID: clusterID,
			},
		},
		"cluster ready, spec with BRN": {
			Spec: clusterfeature.FeatureSpec{
				"clusterDomain": "cluster.org.the.domain",
				"externalDns": obj{
					"domainFilters": arr{
						"",
					},
					"provider": obj{
						"name": "route53",
						"secretId": brn.ResourceName{
							Scheme:         brn.Scheme,
							OrganizationID: orgID,
							ResourceType:   brn.SecretResourceType,
							ResourceID:     providerSecretID,
						}.String(),
						"options": obj{
							"region":    "test-reg",
							"batchSize": 10,
						},
					},
					"txtOwnerId": "my-owner-id",
				},
			},
			Cluster: dummyCluster{
				OrgID:  orgID,
				Status: pkgCluster.Running,
			},
		},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			clusterGetter.Clusters[clusterID] = tc.Cluster

			err := op.Apply(context.Background(), clusterID, tc.Spec)
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
		Clusters: map[uint]dummyCluster{
			clusterID: {
				Status: pkgCluster.Running,
			},
		},
	}
	clusterService := clusterfeatureadapter.NewClusterService(clusterGetter)
	helmService := dummyHelmService{}
	logger := commonadapter.NewNoopLogger()
	op := MakeFeatureOperator(clusterGetter, clusterService, helmService, logger, nil, nil, Config{})

	ctx := context.Background()

	_ = op.Deactivate(ctx, clusterID, nil)
}
