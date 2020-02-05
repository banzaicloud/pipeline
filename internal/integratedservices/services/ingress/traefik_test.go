// Copyright © 2020 Banzai Cloud
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

package ingress

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/pkg/jsonstructure"
)

func TestTraefikManager_CompileChartValues(t *testing.T) {
	orgID := uint(1)

	testCases := map[string]struct {
		Cluster   OperatorCluster
		OrgDomain OrgDomain
		Config    Config
		Spec      Spec
		Expected  interface{}
		Error     interface{}
	}{
		"default config": {
			Cluster: OperatorCluster{
				Cloud:          "azure",
				OrganizationID: orgID,
			},
			OrgDomain: OrgDomain{
				Name:         "my.test.org",
				WildcardName: "*.my.test.org",
			},
			Config: Config{
				Namespace:   "default",
				ReleaseName: "ingress",
				Controllers: []string{"traefik"},
				Charts: ChartsConfig{
					Traefik: TraefikChartConfig{
						Chart:   "stable/traefik",
						Version: "6.6.6",
						Values: map[string]interface{}{
							"ssl": map[string]interface{}{
								"enabled":     true,
								"generateTLS": true,
							},
						},
					},
				},
			},
			Expected: map[string]interface{}{
				"ssl": map[string]interface{}{
					"enabled":        true,
					"generateTLS":    true,
					"defaultCN":      "my.test.org",
					"defaultSANList": []interface{}{"my.test.org", "*.my.test.org"},
				},
			},
		},
		"append amazon lb additional tags to default config": {
			Cluster: OperatorCluster{
				Cloud:          "amazon",
				OrganizationID: orgID,
			},
			OrgDomain: OrgDomain{
				Name:         "my.test.org",
				WildcardName: "*.my.test.org",
			},
			Config: Config{
				Namespace:   "default",
				ReleaseName: "ingress",
				Controllers: []string{"traefik"},
				Charts: ChartsConfig{
					Traefik: TraefikChartConfig{
						Chart:   "stable/traefik",
						Version: "6.6.6",
						Values: map[string]interface{}{
							"ssl": map[string]interface{}{
								"enabled":     true,
								"generateTLS": true,
							},
						},
					},
				},
			},
			Expected: map[string]interface{}{
				"service": map[string]interface{}{
					"annotations": map[string]interface{}{
						"service.beta.kubernetes.io/aws-load-balancer-additional-resource-tags": "banzaicloud-pipeline-managed=true",
					},
				},
				"ssl": map[string]interface{}{
					"enabled":        true,
					"generateTLS":    true,
					"defaultCN":      "my.test.org",
					"defaultSANList": []interface{}{"my.test.org", "*.my.test.org"},
				},
			},
		},
		"append amazon lb additional tags to custom config": {
			Cluster: OperatorCluster{
				Cloud:          "amazon",
				OrganizationID: orgID,
			},
			OrgDomain: OrgDomain{
				Name:         "my.test.org",
				WildcardName: "*.my.test.org",
			},
			Config: Config{
				Namespace:   "default",
				ReleaseName: "ingress",
				Controllers: []string{"traefik"},
				Charts: ChartsConfig{
					Traefik: TraefikChartConfig{
						Chart:   "stable/traefik",
						Version: "6.6.6",
						Values: map[string]interface{}{
							"service": jsonstructure.Object{
								"annotations": jsonstructure.Object{
									"service.beta.kubernetes.io/aws-load-balancer-additional-resource-tags": "foo=bar,fork=spoon",
								},
							},
							"ssl": map[string]interface{}{
								"enabled":     true,
								"generateTLS": true,
							},
						},
					},
				},
			},
			Expected: map[string]interface{}{
				"service": map[string]interface{}{
					"annotations": map[string]interface{}{
						"service.beta.kubernetes.io/aws-load-balancer-additional-resource-tags": "foo=bar,fork=spoon,banzaicloud-pipeline-managed=true",
					},
				},
				"ssl": map[string]interface{}{
					"enabled":        true,
					"generateTLS":    true,
					"defaultCN":      "my.test.org",
					"defaultSANList": []interface{}{"my.test.org", "*.my.test.org"},
				},
			},
		},
	}

	clusterID := uint(1)

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			m := traefikManager{
				clusters: dummyOperatorClusterStore{
					clusters: map[uint]OperatorCluster{
						clusterID: testCase.Cluster,
					},
				},
				config: testCase.Config,
				orgDomainService: dummyOrgDomainService{
					orgs: map[uint]OrgDomain{
						orgID: testCase.OrgDomain,
					},
				},
			}

			values, err := m.compileChartValues(context.Background(), clusterID, testCase.Spec)

			switch testCase.Error {
			case nil, false:
				require.NoError(t, err)
			case true:
				require.Error(t, err)
			default:
				require.Equal(t, testCase.Error, err)
			}

			assert.Equal(t, testCase.Expected, values)
		})
	}
}

type dummyOperatorClusterStore struct {
	clusters map[uint]OperatorCluster
}

func (d dummyOperatorClusterStore) Get(ctx context.Context, clusterID uint) (OperatorCluster, error) {
	if c, ok := d.clusters[clusterID]; ok {
		return c, nil
	}
	return OperatorCluster{}, errors.New("cluster not found")
}

type dummyOrgDomainService struct {
	orgs map[uint]OrgDomain
}

func (d dummyOrgDomainService) GetOrgDomain(ctx context.Context, orgID uint) (OrgDomain, error) {
	if od, ok := d.orgs[orgID]; ok {
		return od, nil
	}
	return OrgDomain{}, errors.New("org not found")
}
