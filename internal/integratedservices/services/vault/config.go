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

package vault

import (
	"emperror.dev/errors"
)

// Config contains configuration for the vault feature.
type Config struct {
	Namespace string
	Managed   ManagedConfig
	Charts    ChartsConfig
}

// ManagedConfig contains cluster managed vault configuration.
type ManagedConfig struct {
	Enabled  bool
	Endpoint string
}

func (c Config) Validate() error {
	if c.Namespace == "" {
		return errors.New("vault namespace is required")
	}

	if c.Managed.Enabled && c.Managed.Endpoint == "" {
		return errors.New("vault endpoint (external address) is required in case of managed vault")
	}

	return nil
}

type ChartsConfig struct {
	Webhook ChartConfig
}

type ChartConfig struct {
	Chart   string
	Version string
	Values  map[string]interface{}
}
