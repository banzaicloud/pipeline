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
	"errors"
	"os"

	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

type vaultIntegratedServiceSpec struct {
	CustomVault CustomVault `json:"customVault" mapstructure:"customVault"`
	Settings    Settings    `json:"settings" mapstructure:"settings"`
}

type CustomVault struct {
	Enabled  bool   `json:"enabled" mapstructure:"enabled"`
	Address  string `json:"address" mapstructure:"address"`
	Policy   string `json:"policy" mapstructure:"policy"`
	SecretID string `json:"secretId" mapstructure:"secretId"`
}

type Settings struct {
	Namespaces      []string `json:"namespaces" mapstructure:"namespaces"`
	ServiceAccounts []string `json:"serviceAccounts" mapstructure:"serviceAccounts"`
}

func bindIntegratedServiceSpec(spec integratedservices.IntegratedServiceSpec) (vaultIntegratedServiceSpec, error) {
	var integratedServiceSpec vaultIntegratedServiceSpec
	if err := mapstructure.Decode(spec, &integratedServiceSpec); err != nil {
		return integratedServiceSpec, integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: integratedServiceName,
			Problem:               "failed to bind integrated service spec",
		}
	}

	return integratedServiceSpec, nil
}

func (s *vaultIntegratedServiceSpec) Validate() error {
	if s.CustomVault.Enabled {
		// address is required in case of custom vault
		if s.CustomVault.Address == "" {
			return errors.New("address field is required in case of custom vault")
		}

		// policy is required in case of custom vault
		if s.CustomVault.Policy == "" && s.CustomVault.SecretID != "" {
			return errors.New("policy field is required in case of custom vault")
		}
	}

	if len(s.Settings.Namespaces) == 1 && s.Settings.Namespaces[0] == "*" &&
		len(s.Settings.ServiceAccounts) == 1 && s.Settings.ServiceAccounts[0] == "*" {
		return errors.New(`both namespaces and service accounts cannot be "*"`)
	}

	return nil
}

func (s *vaultIntegratedServiceSpec) getVaultAddress() (vaultAddress string) {
	if s.CustomVault.Enabled {
		vaultAddress = s.CustomVault.Address
	} else {
		vaultAddress = os.Getenv(vaultAddressEnvKey)
	}
	return
}
