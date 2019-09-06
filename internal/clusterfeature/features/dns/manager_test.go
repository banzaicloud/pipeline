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

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
)

func TestFeatureManager_Name(t *testing.T) {
	mng := MakeFeatureManager(nil, nil, nil)

	assert.Equal(t, "dns", mng.Name())
}

func TestFeatureManager_GetOutput(t *testing.T) {
	clusterID := uint(42)
	clusterName := "the-cluster"

	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]clusterfeatureadapter.Cluster{
			clusterID: dummyCluster{
				Name: clusterName,
			},
		},
	}
	orgDomainService := dummyOrgDomainService{
		Domain: "the.domain",
		OrgID:  13,
	}
	mng := MakeFeatureManager(clusterGetter, nil, orgDomainService)

	ctx := context.Background()

	output, err := mng.GetOutput(ctx, clusterID)

	assert.NoError(t, err)
	assert.Equal(t, clusterfeature.FeatureOutput{
		"autoDns": map[string]interface{}{
			"clusterDomain": "the-cluster.the.domain",
			"zone":          orgDomainService.Domain,
		},
	}, output)
}

func TestFeatureManager_ValidateSpec(t *testing.T) {
	mng := MakeFeatureManager(nil, nil, nil)

	cases := map[string]struct {
		Spec  clusterfeature.FeatureSpec
		Error interface{}
	}{
		"empty spec": {
			Spec:  clusterfeature.FeatureSpec{},
			Error: true,
		},
		"both disabled": {
			Spec: clusterfeature.FeatureSpec{
				"autoDns": obj{
					"enabled": false,
				},
				"customDns": obj{
					"enabled": false,
				},
			},
			Error: true,
		},
		"both enabled": {
			Spec: clusterfeature.FeatureSpec{
				"autoDns": obj{
					"enabled": true,
				},
				"customDns": obj{
					"enabled": true,
				},
			},
			Error: true,
		},
		"autoDns only": {
			Spec: clusterfeature.FeatureSpec{
				"autoDns": obj{
					"enabled": true,
				},
			},
			Error: false,
		},
		"autoDns enabled, customDns disabled": {
			Spec: clusterfeature.FeatureSpec{
				"autoDns": obj{
					"enabled": true,
				},
				"customDns": obj{
					"enabled": false,
				},
			},
		},
		"customDns only": {
			Spec: clusterfeature.FeatureSpec{
				"customDns": obj{
					"enabled": true,
					"domainFilters": arr{
						"",
					},
					"provider": obj{
						"name":     "route53",
						"secretId": "0123456789abcdef",
					},
				},
			},
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

func TestFeatureManager_PrepareSpec(t *testing.T) {
	orgID := uint(42)

	mng := MakeFeatureManager(nil, nil, nil)

	cases := map[string]struct {
		SpecIn  clusterfeature.FeatureSpec
		SpecOut clusterfeature.FeatureSpec
	}{
		"auto DNS enabled": {
			SpecIn: clusterfeature.FeatureSpec{
				"autoDns": obj{
					"enabled": true,
				},
			},
			SpecOut: clusterfeature.FeatureSpec{
				"autoDns": obj{
					"enabled": true,
				},
			},
		},
		"custom DNS enabled": {
			SpecIn: clusterfeature.FeatureSpec{
				"customDns": obj{
					"enabled": true,
					"domainFilters": arr{
						"",
					},
					"provider": obj{
						"name":     "route53",
						"secretId": "0123456789abcdef",
					},
				},
			},
			SpecOut: clusterfeature.FeatureSpec{
				"customDns": obj{
					"enabled": true,
					"domainFilters": arr{
						"",
					},
					"provider": obj{
						"name":     "route53",
						"secretId": "brn:42:secret:0123456789abcdef",
					},
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := auth.SetCurrentOrganizationID(context.Background(), orgID)

			specOut, err := mng.PrepareSpec(ctx, tc.SpecIn)
			assert.NoError(t, err)
			assert.Equal(t, tc.SpecOut, specOut)
		})
	}
}
