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
	"github.com/banzaicloud/pipeline/internal/app/pipeline/cap"
)

// mapCapabilities maps configuration to capabilities.
func mapCapabilities(config configuration) cap.Capabilities {
	caps := cap.Capabilities{
		"features": cap.Cap{
			"vault": cap.Cap{
				"enabled": config.Cluster.Vault.Enabled,
				"managed": config.Cluster.Vault.Managed.Enabled,
			},
			"monitoring": cap.Cap{
				"enabled": config.Cluster.Monitoring.Enabled,
			},
			"logging": cap.Cap{
				"enabled": config.Cluster.Logging.Enabled,
			},
			"dns": cap.Cap{
				"enabled":    config.Cluster.DNS.Enabled,
				"baseDomain": config.Cluster.DNS.BaseDomain,
			},
			"securityScan": cap.Cap{
				"enabled": config.Cluster.SecurityScan.Enabled,
				"managed": config.Cluster.SecurityScan.Anchore.Enabled,
			},
			"expiry": cap.Cap{
				"enabled": config.Cluster.Expiry.Enabled,
			},
			"ingress": cap.Cap{
				"enabled":     config.Cluster.Ingress.Enabled,
				"controllers": config.Cluster.Ingress.Controllers,
			},
		},
		"helm": cap.Cap{
			"version": helmVersion,
		},
	}

	return caps
}
