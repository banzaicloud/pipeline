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

package supported

import (
	"github.com/banzaicloud/pipeline/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// GoogleInfo describes GKE with supported info
type GoogleInfo struct {
	BaseFields
}

// GetType returns cloud type
func (g *GoogleInfo) GetType() string {
	return pkgCluster.Google
}

// GetNameRegexp returns regexp for cluster name
func (g *GoogleInfo) GetNameRegexp() string {
	return pkgCluster.RegexpGKEName
}

// GetLocations returns supported locations
func (g *GoogleInfo) GetLocations() ([]string, error) {
	if len(g.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}
	return cluster.GetZones(g.OrgId, g.SecretId)
}

// GetMachineTypes returns supported machine types
func (g *GoogleInfo) GetMachineTypes() (map[string]pkgCluster.MachineType, error) {
	if len(g.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}
	return cluster.GetAllMachineTypes(g.OrgId, g.SecretId)
}

// GetMachineTypesWithFilter returns supported machine types by location
func (g *GoogleInfo) GetMachineTypesWithFilter(filter *pkgCluster.InstanceFilter) (map[string]pkgCluster.MachineType, error) {

	if len(g.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}

	if len(filter.Location) == 0 {
		return nil, pkgErrors.ErrorRequiredLocation
	}

	return cluster.GetAllMachineTypesByZone(g.OrgId, g.SecretId, filter.Location)
}

// GetKubernetesVersion returns supported k8s versions
func (g *GoogleInfo) GetKubernetesVersion(filter *pkgCluster.KubernetesFilter) (interface{}, error) {

	if len(g.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}

	if filter == nil || len(filter.Location) == 0 {
		return nil, pkgErrors.ErrorRequiredLocation
	}

	return cluster.GetGkeServerConfig(g.OrgId, g.SecretId, filter.Location)
}

// GetImages returns with the supported images (in case of GKE is undefined)
func (g *GoogleInfo) GetImages(filter *pkgCluster.ImageFilter) (map[string][]string, error) {
	return nil, nil
}
