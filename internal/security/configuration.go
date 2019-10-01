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
	Enabled     bool   `json:"enabled" mapstructure:"enabled"`
	Endpoint    string `json:"endpoint" mapstructure:"endpoint"`
	AdminUser   string `json:"adminUser" mapstructure:"adminUser"`
	AdminPass   string `json:"adminPass" mapstructure:"adminPass"`
	AdminSecret string // todo support for secret here?
}

// Validate validates the configuration instance: checks mandatory fields
func (cfg Config) Validate() error {

	if !cfg.Enabled {
		return nil
	}

	_, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return errors.WrapIf(err, "anchore url should be a valid URL")
	}

	if cfg.AdminUser == "" || cfg.AdminPass == "" {
		return errors.New("Both ausername and password values are mandatory")
	}

	return nil
}
