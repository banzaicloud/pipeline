package supported

import (
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/banzai-types/components"
)

type AmazonInfo struct {
	BaseFields
}

// GetType returns cloud type
func (a *AmazonInfo) GetType() string {
	return constants.Amazon
}

// GetNameRegexp returns regexp for cluster name
func (a *AmazonInfo) GetNameRegexp() string {
	return constants.RegexpAWSName
}

// GetLocations returns supported locations
func (a *AmazonInfo) GetLocations() ([]string, error) {
	if regions, err := cluster.ListRegions(); err != nil {
		return nil, err
	} else {
		var locations []string
		for _, region := range regions {
			locations = append(locations, *region.RegionName)
		}
		return locations, nil
	}
}

// GetMachineTypes returns supported machine types
func (a *AmazonInfo) GetMachineTypes() (map[string]components.MachineType, error) {
	return nil, constants.ErrorRequiredZone
}

// GetMachineTypesWithFilter returns supported machine types by location
func (a *AmazonInfo) GetMachineTypesWithFilter(filter *components.InstanceFilter) (map[string]components.MachineType, error) {

	if len(filter.Zone) == 0 {
		return nil, constants.ErrorRequiredZone
	}

	return processMachineTypes(filter.Zone, filter.Tags)
}

// GetKubernetesVersion returns supported k8s versions
func (a *AmazonInfo) GetKubernetesVersion(*components.KubernetesFilter) (interface{}, error) {
	return nil, constants.ErrorCloudInfoK8SNotSupported
}

// processMachineTypes returns supported machine types by region and tags
func processMachineTypes(region string, tags []*string) (map[string]components.MachineType, error) {
	amiList, err := cluster.ListAMIs(region, tags)
	if err != nil {
		return nil, err
	}

	response := make(map[string]components.MachineType)
	var amiSlice []string
	for _, ami := range amiList {
		amiSlice = append(amiSlice, *ami.ImageId)
	}

	response[region] = amiSlice

	return response, nil
}
