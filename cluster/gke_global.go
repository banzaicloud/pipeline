// Copyright © 2018 Banzai Cloud
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

package cluster

import (
	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/providers/google"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	gke "google.golang.org/api/container/v1"
)

// GetGkeServerConfig returns all supported K8S versions
func GetGkeServerConfig(orgId uint, secretId, zone string) (*gke.ServerConfig, error) {
	g := GKECluster{
		model: &google.GKEClusterModel{
			Cluster: cluster.ClusterModel{
				OrganizationID: orgId,
				SecretID:       secretId,
				Cloud:          pkgCluster.Google,
			},
		},
	}
	return g.GetGkeServerConfig(zone)
}

// GetAllMachineTypesByZone returns all supported machine type by zone
func GetAllMachineTypesByZone(orgId uint, secretId, zone string) (map[string]pkgCluster.MachineTypes, error) {
	g := &GKECluster{
		model: &google.GKEClusterModel{
			Cluster: cluster.ClusterModel{
				OrganizationID: orgId,
				SecretID:       secretId,
				Cloud:          pkgCluster.Google,
			},
		},
	}
	return g.GetAllMachineTypesByZone(zone)
}

// GetAllMachineTypes returns all supported machine types
func GetAllMachineTypes(orgId uint, secretId string) (map[string]pkgCluster.MachineTypes, error) {
	g := &GKECluster{
		model: &google.GKEClusterModel{
			Cluster: cluster.ClusterModel{
				OrganizationID: orgId,
				SecretID:       secretId,
				Cloud:          pkgCluster.Google,
			},
		},
	}

	return g.GetAllMachineTypes()
}

// GetZones lists all supported zones
func GetZones(orgId uint, secretId string) ([]string, error) {
	g := &GKECluster{
		model: &google.GKEClusterModel{
			Cluster: cluster.ClusterModel{
				OrganizationID: orgId,
				SecretID:       secretId,
				Cloud:          pkgCluster.Google,
			},
		},
	}
	return g.GetZones()
}
