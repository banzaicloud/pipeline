package supported

import (
	"github.com/banzaicloud/pipeline/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// AzureInfo describes AKS with supported info
type AzureInfo struct {
	BaseFields
}

// GetType returns cloud type
func (a *AzureInfo) GetType() string {
	return pkgCluster.Azure
}

// GetNameRegexp returns regexp for cluster name
func (a *AzureInfo) GetNameRegexp() string {
	return pkgCluster.RegexpAKSName
}

// GetLocations returns supported locations
func (a *AzureInfo) GetLocations() ([]string, error) {
	return cluster.GetLocations(a.OrgId, a.SecretId)
}

// GetMachineTypes returns supported machine types
func (a *AzureInfo) GetMachineTypes() (map[string]pkgCluster.MachineType, error) {
	return nil, pkgErrors.ErrorRequiredLocation
}

// GetMachineTypesWithFilter returns supported machine types by location
func (a *AzureInfo) GetMachineTypesWithFilter(filter *pkgCluster.InstanceFilter) (map[string]pkgCluster.MachineType, error) {

	if len(filter.Location) == 0 {
		return nil, pkgErrors.ErrorRequiredLocation
	}

	return cluster.GetMachineTypes(a.OrgId, a.SecretId, filter.Location)
}

// GetKubernetesVersion returns supported k8s versions
func (a *AzureInfo) GetKubernetesVersion(filter *pkgCluster.KubernetesFilter) (interface{}, error) {

	if filter == nil || len(filter.Location) == 0 {
		return nil, pkgErrors.ErrorRequiredLocation
	}

	return cluster.GetKubernetesVersion(a.OrgId, a.SecretId, filter.Location)
}

// GetImages returns with the supported images (in case of AKS is undefined)
func (a *AzureInfo) GetImages(filter *pkgCluster.ImageFilter) (map[string][]string, error) {
	return nil, nil
}
