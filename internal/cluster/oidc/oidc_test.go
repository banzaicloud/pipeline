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

package oidc_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	clusterAuth "github.com/banzaicloud/pipeline/internal/cluster/auth"
	"github.com/banzaicloud/pipeline/internal/cluster/oidc"
	"github.com/banzaicloud/pipeline/src/auth"
)

const (
	oidcIssuer   = "myDexURL.com/dex"
	clusterID    = 1
	clientID     = "clientID"
	clientSecret = "clientSecret"
)

type dummyClusterClientSecretGetter struct {
}

func (dummyClusterClientSecretGetter) GetClusterClientSecret(_ context.Context, _ uint) (clusterAuth.ClusterClientSecret, error) {
	return clusterAuth.ClusterClientSecret{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}, nil
}

func TestOIDC_CreateNewOIDCResponse(t *testing.T) {
	dummyAuthConfig := auth.OIDCConfig{
		Issuer: oidcIssuer,
	}

	testCases := []struct {
		name             string
		oidcEnabled      bool
		expectedResponse oidc.OIDC
	}{
		{
			name:        "generate OIDC response (enabled)",
			oidcEnabled: true,
			expectedResponse: oidc.OIDC{
				Enabled:      true,
				IdpURL:       oidcIssuer,
				ClientSecret: clientSecret,
				ClientID:     clientID,
			},
		},
		{
			name:        "generate OIDC response (disabled)",
			oidcEnabled: false,
			expectedResponse: oidc.OIDC{
				Enabled: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			creator := oidc.NewCreator(dummyAuthConfig, dummyClusterClientSecretGetter{})

			response, err := creator.CreateNewOIDCResponse(context.TODO(), tc.oidcEnabled, clusterID)

			assert.NoError(t, err)
			assert.Equal(t, response, tc.expectedResponse)
		})
	}
}
