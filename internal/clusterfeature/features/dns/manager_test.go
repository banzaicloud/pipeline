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
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/src/auth"
)

func TestFeatureManager_Name(t *testing.T) {
	mng := NewFeatureManager(nil, nil, Config{})

	assert.Equal(t, "dns", mng.Name())
}

func TestFeatureManager_GetOutput(t *testing.T) {
	clusterID := uint(42)
	version := "1.2.3"

	mng := NewFeatureManager(nil, nil, Config{
		Charts: ChartsConfig{
			ExternalDNS: ExternalDNSChartConfig{
				ChartConfigBase: ChartConfigBase{
					Version: version,
				},
			},
		},
	})

	output, err := mng.GetOutput(context.Background(), clusterID, nil)

	assert.NoError(t, err)
	assert.Equal(t, clusterfeature.FeatureOutput{
		"externalDns": map[string]interface{}{
			"version": version,
		},
	}, output)
}

func TestFeatureManager_ValidateSpec_ValidSpec(t *testing.T) {
	mng := NewFeatureManager(nil, nil, Config{})

	spec := clusterfeature.FeatureSpec{
		"clusterDomain": "cluster.org.my.domain",
		"externalDns": obj{
			"domainFilters": arr{
				"",
			},
			"policy": "sync*|upsert-only",
			"provider": obj{
				"name":     "route53",
				"secretID": "0123456789abcdef",
			},
			"sources": arr{
				"ingress",
			},
			"txtOwnerId": "my-owner-id",
		},
	}

	err := mng.ValidateSpec(context.Background(), spec)
	require.NoError(t, err)
}
func TestFeatureManager_ValidateSpec_InvalidSpec(t *testing.T) {
	mng := NewFeatureManager(nil, nil, Config{})

	err := mng.ValidateSpec(context.Background(), clusterfeature.FeatureSpec{})
	require.Error(t, err)

	var e clusterfeature.InvalidFeatureSpecError
	assert.True(t, errors.As(err, &e))
}

func TestFeatureManager_PrepareSpec(t *testing.T) {
	orgID := uint(42)
	clusterID := uint(13)
	clusterUID := "ca951029-208d-4cb1-87fe-6e7369d32949"

	mng := NewFeatureManager(
		dummyClusterOrgIDGetter{
			Mapping: map[uint]uint{
				clusterID: orgID,
			},
		},
		dummyClusterUIDGetter{
			Mapping: map[uint]string{
				clusterID: clusterUID,
			},
		},
		Config{},
	)

	cases := map[string]struct {
		SpecIn  clusterfeature.FeatureSpec
		SpecOut clusterfeature.FeatureSpec
	}{
		"provider with secret, without txtOwnerID": {
			SpecIn: clusterfeature.FeatureSpec{
				"externalDns": obj{
					"provider": obj{
						"secretId":         "0123456789abcdef",
						"some-other-field": "some-value",
					},
				},
			},
			SpecOut: clusterfeature.FeatureSpec{
				"externalDns": obj{
					"provider": obj{
						"secretId":         "brn:42:secret:0123456789abcdef",
						"some-other-field": "some-value",
					},
					"txtOwnerId": clusterUID,
				},
			},
		},
		"provider without secret, with txtOwnerID": {
			SpecIn: clusterfeature.FeatureSpec{
				"externalDns": obj{
					"provider": obj{
						"some-other-field": "some-value",
					},
					"txtOwnerId": "my-owner-id",
				},
			},
			SpecOut: clusterfeature.FeatureSpec{
				"externalDns": obj{
					"provider": obj{
						"some-other-field": "some-value",
					},
					"txtOwnerId": "my-owner-id",
				},
			},
		},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			ctx := auth.SetCurrentOrganizationID(context.Background(), orgID)

			specOut, err := mng.PrepareSpec(ctx, clusterID, tc.SpecIn)
			assert.NoError(t, err)
			assert.Equal(t, tc.SpecOut, specOut)
		})
	}
}

type dummyClusterOrgIDGetter struct {
	Mapping map[uint]uint
}

func (d dummyClusterOrgIDGetter) GetClusterOrgID(_ context.Context, clusterID uint) (uint, error) {
	if orgID, ok := d.Mapping[clusterID]; ok {
		return orgID, nil
	}
	return 0, errors.New("cluster not found")
}

type dummyClusterUIDGetter struct {
	Mapping map[uint]string
}

func (d dummyClusterUIDGetter) GetClusterUID(_ context.Context, clusterID uint) (string, error) {
	if uid, ok := d.Mapping[clusterID]; ok {
		return uid, nil
	}
	return "", errors.New("cluster not found")
}
