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
	"net/url"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/anchore"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
)

type Config struct {
	Anchore           AnchoreConfig
	PipelineNamespace string
	Webhook           WebhookConfig
}

func (c Config) Validate() error {
	return errors.Combine(c.Anchore.Validate())
}

type AnchoreConfig struct {
	Enabled        bool
	anchore.Config `mapstructure:",squash"`
}

func (c AnchoreConfig) Validate() error {
	var err error

	if c.Enabled {
		_, e := url.Parse(c.Endpoint)
		err = errors.Append(err, errors.Wrap(e, "anchore endpoint must be a valid URL"))

		if c.User == "" {
			err = errors.Append(err, errors.New("anchore user is required"))
		}

		if c.Password == "" {
			err = errors.Append(err, errors.New("anchore password is required"))
		}
	}

	return err
}

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
	insecure          bool
}

// NewClusterAnchoreConfigProvider returns a new ClusterAnchoreConfigProvider.
func NewClusterAnchoreConfigProvider(
	endpoint string,
	userNameGenerator UserNameGenerator,
	userSecretStore UserSecretStore,
	insecure bool,
) ClusterAnchoreConfigProvider {
	return ClusterAnchoreConfigProvider{
		endpoint:          endpoint,
		userNameGenerator: userNameGenerator,
		userSecretStore:   userSecretStore,
		insecure:          insecure,
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
		Insecure: p.insecure,
	}, nil
}

// CustomAnchoreConfigProvider returns custom Anchore configuration for a cluster.
type CustomAnchoreConfigProvider struct {
	integratedServicesRepository integratedservices.IntegratedServiceRepository
	secretStore                  services.SecretStore

	logger services.Logger
}

// NewCustomAnchoreConfigProvider returns a new ConfigProvider.
func NewCustomAnchoreConfigProvider(
	integratedServiceRepository integratedservices.IntegratedServiceRepository,
	secretStore services.SecretStore,

	logger services.Logger,
) CustomAnchoreConfigProvider {
	return CustomAnchoreConfigProvider{
		integratedServicesRepository: integratedServiceRepository,
		secretStore:                  secretStore,

		logger: logger,
	}
}

// GetConfiguration returns Anchore configuration for a cluster.
func (p CustomAnchoreConfigProvider) GetConfiguration(ctx context.Context, clusterID uint) (anchore.Config, error) {
	logger := p.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID})

	integratedService, err := p.integratedServicesRepository.GetIntegratedService(ctx, clusterID, IntegratedServiceName)
	if err != nil {
		return anchore.Config{}, err
	}

	spec, err := bindIntegratedServiceSpec(integratedService.Spec)
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
		Endpoint:   spec.CustomAnchore.Url,
		User:       secret[secrettype.Username],
		Password:   secret[secrettype.Password],
		Insecure:   spec.CustomAnchore.Insecure,
		PolicyPath: spec.CustomAnchore.PolicyPath,
	}, nil
}

// WebhookConfig encapsulates configuration of the image validator webhook
// sensitive defaults provided through env vars
type WebhookConfig struct {
	Chart     string
	Version   string
	Release   string
	Namespace string
	Values    map[string]interface{}
}
