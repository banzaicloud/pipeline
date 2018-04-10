package supported

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/cluster"
)

type GoogleInfo struct {
	BaseFields
}

// GetType returns cloud type
func (g *GoogleInfo) GetType() string {
	return constants.Google
}

// GetNameRegexp returns regexp for cluster name
func (g *GoogleInfo) GetNameRegexp() string {
	return constants.RegexpGKEName
}

// GetLocations returns supported locations
func (g *GoogleInfo) GetLocations() ([]string, error) {
	if len(g.SecretId) == 0 {
		return nil, constants.ErrorRequiredSecretId
	}
	return cluster.GetZones(g.OrgId, g.SecretId)
}

// GetLocations returns supported machine types
func (g *GoogleInfo) GetMachineTypes() (map[string]components.MachineType, error) {
	if len(g.SecretId) == 0 {
		return nil, constants.ErrorRequiredSecretId
	}
	return cluster.GetAllMachineTypes(g.OrgId, g.SecretId)
}

// GetMachineTypesWithFilter returns supported machine types by location
func (g *GoogleInfo) GetMachineTypesWithFilter(filter *components.InstanceFilter) (map[string]components.MachineType, error) {

	if len(g.SecretId) == 0 {
		return nil, constants.ErrorRequiredSecretId
	}

	if len(filter.Zone) == 0 {
		return nil, constants.ErrorRequiredZone
	}

	return cluster.GetAllMachineTypesByZone(g.OrgId, g.SecretId, filter.Zone)
}

// GetKubernetesVersion returns supported k8s versions
func (g *GoogleInfo) GetKubernetesVersion(filter *components.KubernetesFilter) (interface{}, error) {

	if len(g.SecretId) == 0 {
		return nil, constants.ErrorRequiredSecretId
	}

	if filter == nil || len(filter.Zone) == 0 {
		return nil, constants.ErrorRequiredZone
	}

	return cluster.GetGkeServerConfig(g.OrgId, g.SecretId, filter.Zone)
}
