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
	"net/url"

	"emperror.dev/errors"
)

// Config struct holding anchore related configuration
type Config struct {
	//ApiEnabled drives the "security scan capability"; expected to be true most of the time
	ApiEnabled bool   `json:"apiEnabled" mapstructure:"apiEnabled"`
	Enabled    bool   `json:"enabled" mapstructure:"enabled"`
	Endpoint   string `json:"endpoint" mapstructure:"endpoint"`
	AdminUser  string `json:"adminUser" mapstructure:"adminUser"`
	AdminPass  string `json:"adminPass" mapstructure:"adminPass"`
	// this is populated by the configuration service (custom anchore case!)
	UserSecret string `json:"secretId" mapstructure:"secretId"`
}

// Validate validates the configuration instance: checks mandatory fields
func (cfg Config) Validate() error {
	// the whole security api is disabled!
	if !cfg.ApiEnabled {
		return nil
	}

	if !cfg.Enabled {
		return nil
	}

	_, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return errors.WrapIf(err, "anchore url should be a valid URL")
	}

	if cfg.AdminUser == "" || cfg.AdminPass == "" {
		return errors.New("Both username and password values are mandatory")
	}

	return nil
}
