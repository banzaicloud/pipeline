// Copyright © 2021 Banzai Cloud
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

package helm

import (
	"fmt"

	"github.com/banzaicloud/pipeline/pkg/sdk/semver"
)

// ChartConfig describes a Helm chart configuration.
type ChartConfig struct {
	Name       string                 `json:"name" mapstructure:"name" yaml:"name"`
	Version    string                 `json:"version" mapstructure:"version" yaml:"version"`
	Repository string                 `json:"repository" mapstructure:"repository" yaml:"repository"`
	Values     map[string]interface{} `json:"values,omitempty" mapstructure:"values,omitempty" yaml:"values,omitempty"`
}

// IsLessThan determines whether the receiver is ordered less than the specified
// chart config.
func (config ChartConfig) IsLessThan(otherConfig ChartConfig) bool {
	if config.Name != otherConfig.Name {
		return config.Name < otherConfig.Name
	} else if config.Version != otherConfig.Version {
		configVersion, err := semver.NewVersionFromString(config.Version)
		if err != nil { // Note: not a semantic version, comparing alphabetically.
			return config.Version < otherConfig.Version
		}

		otherConfigVersion, err := semver.NewVersionFromString(otherConfig.Version)
		if err != nil { // Note: not a semantic version, comparing alphabetically.
			return config.Version < otherConfig.Version
		}

		return configVersion.IsLessThan(otherConfigVersion)
	} else if config.Repository != otherConfig.Repository {
		return config.Repository < otherConfig.Repository
	}

	return fmt.Sprint(config.Values) < fmt.Sprint(otherConfig.Values)
}
