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
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/mitchellh/mapstructure"
)

type vaultFeatureSpec struct {
	CustomVault CustomVault `json:"customVault"`
	Settings    Settings    `json:"settings"`
}

type CustomVault struct {
	Enabled bool   `json:"enabled"`
	Address string `json:"address"`
	Token   string `json:"token"`
}

type Settings struct {
	Namespaces      []string `json:"namespaces"`
	ServiceAccounts []string `json:"serviceAccounts"`
}

func bindFeatureSpec(spec clusterfeature.FeatureSpec) (vaultFeatureSpec, error) {
	var featureSpec vaultFeatureSpec
	if err := mapstructure.Decode(spec, &featureSpec); err != nil {
		return featureSpec, clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     "failed to bind feature spec",
		}
	}

	return featureSpec, nil
}
