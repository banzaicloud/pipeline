package supported

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/cluster"
)

type AzureInfo struct {
	BaseFields
}

// GetType returns cloud type
func (a *AzureInfo) GetType() string {
	return constants.Azure
}

// GetNameRegexp returns regexp for cluster name
func (a *AzureInfo) GetNameRegexp() string {
	return constants.RegexpAKSName
}

// GetLocations returns supported locations
func (a *AzureInfo) GetLocations() ([]string, error) {
	return cluster.GetLocations(a.OrgId, a.SecretId)
}

// GetMachineTypes returns supported machine types
func (a *AzureInfo) GetMachineTypes() (map[string]components.MachineType, error) {
	return nil, constants.ErrorRequiredZone
}

// GetMachineTypesWithFilter returns supported machine types by location
func (a *AzureInfo) GetMachineTypesWithFilter(filter *components.InstanceFilter) (map[string]components.MachineType, error) {

	if len(filter.Zone) == 0 {
		return nil, constants.ErrorRequiredZone
	}

	return cluster.GetMachineTypes(a.OrgId, a.SecretId, filter.Zone)
}

// GetKubernetesVersion returns supported k8s versions
func (a *AzureInfo) GetKubernetesVersion(filter *components.KubernetesFilter) (interface{}, error) {

	if filter == nil || len(filter.Zone) == 0 {
		return nil, constants.ErrorRequiredZone
	}

	return cluster.GetKubernetesVersion(a.OrgId, a.SecretId, filter.Zone)
}
