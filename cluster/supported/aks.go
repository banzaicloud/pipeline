package supported

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/cluster"
)

// AzureInfo describes AKS with supported info
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
	return nil, constants.ErrorRequiredLocation
}

// GetMachineTypesWithFilter returns supported machine types by location
func (a *AzureInfo) GetMachineTypesWithFilter(filter *components.InstanceFilter) (map[string]components.MachineType, error) {

	if len(filter.Location) == 0 {
		return nil, constants.ErrorRequiredLocation
	}

	return cluster.GetMachineTypes(a.OrgId, a.SecretId, filter.Location)
}

// GetKubernetesVersion returns supported k8s versions
func (a *AzureInfo) GetKubernetesVersion(filter *components.KubernetesFilter) (interface{}, error) {

	if filter == nil || len(filter.Location) == 0 {
		return nil, constants.ErrorRequiredLocation
	}

	return cluster.GetKubernetesVersion(a.OrgId, a.SecretId, filter.Location)
}

// GetImages returns with the supported images (in case of AKS is undefined)
func (a *AzureInfo) GetImages(filter *components.ImageFilter) (map[string][]string, error) {
	return nil, nil
}
