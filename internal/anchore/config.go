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

package anchore

import (
	"context"

	"emperror.dev/errors"
)

// Config holds configuration required for connecting the Anchore API.
type Config struct {
	Endpoint string
	User     string
	Password string
}

// ErrConfigNotFound is returned by config providers to indicate it couldn't find any configuration.
const ErrConfigNotFound = errors.Sentinel("anchore config not found")

// ConfigProvider returns Anchore configuration for a cluster.
type ConfigProvider interface {
	// GetConfiguration returns Anchore configuration for a cluster.
	GetConfiguration(ctx context.Context, clusterID uint) (Config, error)
}

// ConfigProviderChain loops through a list of providers to fetch a config.
type ConfigProviderChain []ConfigProvider

func (c ConfigProviderChain) GetConfiguration(ctx context.Context, clusterID uint) (Config, error) {
	for _, provider := range c {
		config, err := provider.GetConfiguration(ctx, clusterID)
		if err != nil {
			if errors.Is(err, ErrConfigNotFound) {
				continue
			}

			return Config{}, err
		}

		return config, nil
	}

	return Config{}, ErrConfigNotFound
}

// StaticConfigProvider returns static configuration.
type StaticConfigProvider struct {
	Config Config
}

func (p StaticConfigProvider) GetConfiguration(ctx context.Context, clusterID uint) (Config, error) {
	return p.Config, nil
}
