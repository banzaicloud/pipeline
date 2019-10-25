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

	"github.com/banzaicloud/pipeline/internal/anchore"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
)

// UserNameGenerator generates an Anchore username for a cluster.
type UserNameGenerator interface {
	// GenerateUsername generates an Anchore username for a cluster.
	GenerateUsername(ctx context.Context, clusterID uint) (string, error)
}

// UserSecretStore stores Anchore user secrets.
type UserSecretStore interface {
	// GetPasswordForUser returns the password for a user.
	GetPasswordForUser(ctx context.Context, userName string) (string, error)
}

// ClusterAnchoreConfigProvider returns static configuration.
type ClusterAnchoreConfigProvider struct {
	endpoint          string
	userNameGenerator UserNameGenerator
	userSecretStore   UserSecretStore
}

// NewClusterAnchoreConfigProvider returns a new ClusterAnchoreConfigProvider.
func NewClusterAnchoreConfigProvider(
	endpoint string,
	userNameGenerator UserNameGenerator,
	userSecretStore UserSecretStore,
) ClusterAnchoreConfigProvider {
	return ClusterAnchoreConfigProvider{
		endpoint:          endpoint,
		userNameGenerator: userNameGenerator,
		userSecretStore:   userSecretStore,
	}
}

func (p ClusterAnchoreConfigProvider) GetConfiguration(ctx context.Context, clusterID uint) (anchore.Config, error) {
	userName, err := p.userNameGenerator.GenerateUsername(ctx, clusterID)
	if err != nil {
		return anchore.Config{}, err
	}

	password, err := p.userSecretStore.GetPasswordForUser(ctx, userName)
	if err != nil {
		if errors.As(err, &common.SecretNotFoundError{}) {
			return anchore.Config{}, anchore.ErrConfigNotFound
		}

		return anchore.Config{}, err
	}

	return anchore.Config{
		Endpoint: p.endpoint,
		User:     userName,
		Password: password,
	}, nil
}

// CustomAnchoreConfigProvider returns custom Anchore configuration for a cluster.
type CustomAnchoreConfigProvider struct {
	featureRepository clusterfeature.FeatureRepository
	secretStore       features.SecretStore

	logger features.Logger
}

// NewCustomAnchoreConfigProvider returns a new ConfigProvider.
func NewCustomAnchoreConfigProvider(
	featureRepository clusterfeature.FeatureRepository,
	secretStore features.SecretStore,

	logger features.Logger,
) CustomAnchoreConfigProvider {
	return CustomAnchoreConfigProvider{
		featureRepository: featureRepository,
		secretStore:       secretStore,

		logger: logger,
	}
}

// GetConfiguration returns Anchore configuration for a cluster.
func (p CustomAnchoreConfigProvider) GetConfiguration(ctx context.Context, clusterID uint) (anchore.Config, error) {
	logger := p.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID})

	feature, err := p.featureRepository.GetFeature(ctx, clusterID, FeatureName)
	if err != nil {
		return anchore.Config{}, err
	}

	spec, err := bindFeatureSpec(feature.Spec)
	if err != nil {
		return anchore.Config{}, err
	}

	if !spec.CustomAnchore.Enabled {
		logger.Debug("no custom anchore config found for cluster")

		return anchore.Config{}, anchore.ErrConfigNotFound
	}

	secret, err := p.secretStore.GetSecretValues(ctx, spec.CustomAnchore.SecretID)
	if err != nil {
		return anchore.Config{}, err
	}

	return anchore.Config{
		Endpoint: spec.CustomAnchore.Url,
		User:     secret[secrettype.Username],
		Password: secret[secrettype.Password],
	}, nil
}
