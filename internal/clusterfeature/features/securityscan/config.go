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

package securityscan

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/internal/securityscan"
)

// AnchoreConfigProvider returns Anchore configuration for a cluster.
type AnchoreConfigProvider struct {
	config            *securityscan.AnchoreConfig
	featureRepository clusterfeature.FeatureRepository
	secretStore       features.SecretStore

	logger features.Logger
}

// NewAnchoreConfigProvider returns a new AnchoreConfigProvider.
func NewAnchoreConfigProvider(
	config *securityscan.AnchoreConfig,
	featureRepository clusterfeature.FeatureRepository,
	secretStore features.SecretStore,

	logger features.Logger,
) AnchoreConfigProvider {
	return AnchoreConfigProvider{
		config:            config,
		featureRepository: featureRepository,
		secretStore:       secretStore,

		logger: logger,
	}
}

// GetConfiguration returns Anchore configuration for a cluster.
func (p AnchoreConfigProvider) GetConfiguration(ctx context.Context, clusterID uint) (securityscan.AnchoreConfig, error) {
	logger := p.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID})

	feature, err := p.featureRepository.GetFeature(ctx, clusterID, FeatureName)
	if err != nil {
		return securityscan.AnchoreConfig{}, err
	}

	spec, err := bindFeatureSpec(feature.Spec)
	if err != nil {
		return securityscan.AnchoreConfig{}, err
	}

	if !spec.CustomAnchore.Enabled {
		logger.Debug("no custom anchore config found for cluster")

		if p.config == nil {
			return securityscan.AnchoreConfig{}, errors.New("no custom or global anchore config found")
		}

		return *p.config, nil
	}

	secret, err := p.secretStore.GetSecretValues(ctx, spec.CustomAnchore.SecretID)
	if err != nil {
		return securityscan.AnchoreConfig{}, err
	}

	return securityscan.AnchoreConfig{
		Endpoint: spec.CustomAnchore.Url,
		User:     secret[secrettype.Username],
		Password: secret[secrettype.Password],
	}, nil
}
