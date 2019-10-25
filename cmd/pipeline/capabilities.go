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

package main

import (
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/cap"
)

// mapCapabilities maps configuration to capabilities.
func mapCapabilities(config configuration) cap.Capabilities {
	return cap.Capabilities{
		"cicd": cap.Cap{
			"enabled": viper.GetBool("cicd.enabled"),
		},
		"issue": cap.Cap{
			"enabled": config.Frontend.Issue.Enabled,
		},
		"features": cap.Cap{
			"vault": cap.Cap{
				"enabled": config.Cluster.Vault.Enabled,
				"managed": config.Cluster.Vault.Managed.Enabled,
			},
			"monitoring": cap.Cap{
				"enabled": config.Cluster.Monitoring.Enabled,
			},
			"securityScan": cap.Cap{
				"enabled": config.Cluster.SecurityScan.Enabled,
				"managed": config.Cluster.SecurityScan.Managed.Enabled,
			},
		},
	}
}
