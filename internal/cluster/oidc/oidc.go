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

package oidc

import (
	"context"

	"emperror.dev/errors"

	clusterAuth "github.com/banzaicloud/pipeline/internal/cluster/auth"
	"github.com/banzaicloud/pipeline/src/auth"
)

type Creator struct {
	config       auth.OIDCConfig
	secretGetter clusterAuth.ClusterClientSecretGetter
}

type OIDC struct {
	Enabled      bool
	IdpURL       string
	ClientID     string
	ClientSecret string
}

func NewCreator(config auth.OIDCConfig, secretGetter clusterAuth.ClusterClientSecretGetter) *Creator {
	return &Creator{
		config:       config,
		secretGetter: secretGetter,
	}
}

func (c *Creator) CreateNewOIDCResponse(context context.Context, isEnabled bool, clusterID uint) (OIDC, error) {
	if isEnabled {
		var secret clusterAuth.ClusterClientSecret
		secret, secretError := c.secretGetter.GetClusterClientSecret(context, clusterID)
		if secretError != nil {
			return OIDC{}, errors.WrapIf(secretError, "error getting cluster client secret")
		}

		return OIDC{
			Enabled:      isEnabled,
			IdpURL:       c.config.Issuer,
			ClientID:     secret.ClientID,
			ClientSecret: secret.ClientSecret,
		}, nil
	}

	return OIDC{Enabled: false}, nil
}
