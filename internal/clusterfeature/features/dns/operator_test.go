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

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/dns/route53"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/pkg/brn"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

func TestFeatureOperator_Name(t *testing.T) {
	op := MakeFeatureOperator(nil, nil, nil, nil, nil, nil)

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
						pkgSecret.AwsRegion:          "us-west-2",
						pkgSecret.AwsAccessKeyId:     "my-access-key-id",
						pkgSecret.AwsSecretAccessKey: "my-access-key-secret",
					},
				},
				providerSecretID: {
					ID:   providerSecretID,
					Name: providerSecretName,
					Type: pkgCluster.Amazon,
					Values: map[string]string{
						pkgSecret.AwsRegion:          "moon-21",
						pkgSecret.AwsAccessKeyId:     "an-access-key-id",
						pkgSecret.AwsSecretAccessKey: "an-access-key-secret",
					},
				},
			},
		},
	}
	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]clusterfeatureadapter.Cluster{},
	}
	clusterService := clusterfeatureadapter.NewClusterService(clusterGetter)
	helmService := dummyHelmService{}
	logger := commonadapter.NewNoopLogger()
	orgDomainService := dummyOrgDomainService{
		Domain: "the.domain",
		OrgID:  orgID,
	}
	secretStore := commonadapter.NewSecretStore(orgSecretStore, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
	op := MakeFeatureOperator(clusterGetter, clusterService, helmService, logger, orgDomainService, secretStore)

	cases := map[string]struct {
		Spec    clusterfeature.FeatureSpec
		Cluster clusterfeatureadapter.Cluster
		Error   interface{}
	}{
		"auto DNS, cluster ready": {
			Spec: clusterfeature.FeatureSpec{
				"autoDns": obj{
					"enabled": true,
				},
			},
			Cluster: dummyCluster{
				OrgID: orgID,
				Ready: true,
			},
		},
		"auto DNS, cluster not ready": {
			Spec: clusterfeature.FeatureSpec{
				"autoDns": obj{
					"enabled": true,
				},
			},
			Cluster: dummyCluster{
				OrgID: orgID,
				Ready: false,
			},
			Error: clusterfeature.ClusterIsNotReadyError{
				ClusterID: clusterID,
			},
		},
		"custom DNS, cluster ready": {
			Spec: clusterfeature.FeatureSpec{
				"customDns": obj{
					"enabled": true,
					"domainFilters": arr{
						"",
					},
					"provider": obj{
						"name":     "route53",
						"secretId": providerSecretID,
					},
				},
			},
			Cluster: dummyCluster{
				OrgID: orgID,
				Ready: true,
			},
		},
		"custom DNS, cluster ready, with BRN": {
			Spec: clusterfeature.FeatureSpec{
				"customDns": obj{
					"enabled": true,
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
					},
				},
			},
			Cluster: dummyCluster{
				OrgID: orgID,
				Ready: true,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			clusterGetter.Clusters[clusterID] = tc.Cluster

			ctx := context.Background()

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
	logger := commonadapter.NewNoopLogger()
	op := MakeFeatureOperator(clusterGetter, clusterService, helmService, logger, nil, nil)

	ctx := context.Background()

	_ = op.Deactivate(ctx, clusterID)
}
