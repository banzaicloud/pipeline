package supported

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/cluster"
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
	return nil, constants.ErrorRequiredLocation
}

// GetMachineTypesWithFilter returns supported machine types by location
func (a *AmazonInfo) GetMachineTypesWithFilter(filter *components.InstanceFilter) (map[string]components.MachineType, error) {
	// todo NOTE: until aws sdk dont support getting instance types
	response := make(map[string]components.MachineType)
	response[filter.Location] = []string{
		"t2.nano",
		"t2.micro",
		"t2.small",
		"t2.medium",
		"t2.large",
		"t2.xlarge",
		"t2.2xlarge",
		"m5.large",
		"m5.xlarge",
		"m5.2xlarge",
		"m5.4xlarge",
		"m5.12xlarge",
		"m5.24xlarge",
		"m4.large",
		"m4.xlarge",
		"m4.2xlarge",
		"m4.4xlarge",
		"m4.10xlarge",
		"m4.16xlarge",
		"c5.large",
		"c5.xlarge",
		"c5.2xlarge",
		"c5.4xlarge",
		"c5.9xlarge",
		"c5.18xlarge",
		"c4.large",
		"c4.xlarge",
		"c4.2xlarge",
		"c4.4xlarge",
		"c4.8xlarge",
		"g3.4xlarge",
		"g3.16xlarge",
		"p2.xlarge",
		"p2.8xlarge",
		"p2.16xlarge",
		"p3.2xlarge",
		"p3.8xlarge",
		"p3.16xlarge",
		"r4.large",
		"r4.xlarge",
		"r4.2xlarge",
		"r4.4xlarge",
		"r4.8xlarge",
		"r4.16xlarge",
		"x1.16xlarge",
		"x1.32xlarge",
		"d2.xlarge",
		"d2.2xlarge",
		"d2.4xlarge",
		"d2.8xlarge",
		"i2.xlarge",
		"i2.2xlarge",
		"i2.4xlarge",
		"i2.8xlarge",
		"h1.2xlarge",
		"h1.4xlarge",
		"h1.8xlarge",
		"h1.16xlarge",
		"i3.large",
		"i3.xlarge",
		"i3.2xlarge",
		"i3.4xlarge",
		"i3.8xlarge",
		"i3.16xlarge",
	}
	return response, nil
}

// GetKubernetesVersion returns supported k8s versions
func (a *AmazonInfo) GetKubernetesVersion(*components.KubernetesFilter) (interface{}, error) {
	return nil, constants.ErrorCloudInfoK8SNotSupported
}

// processAMIList returns supported AMIs by region and tags
func processAMIList(region string, tags []*string) (map[string][]string, error) {
	amiList, err := cluster.ListAMIs(region, tags)
	if err != nil {
		return nil, err
	}

	response := make(map[string][]string)
	var amiSlice []string
	for _, ami := range amiList {
		amiSlice = append(amiSlice, *ami.ImageId)
	}

	response[region] = amiSlice

	return response, nil
}

// GetImages returns supported AMIs
func (a *AmazonInfo) GetImages(filter *components.ImageFilter) (map[string][]string, error) {
	if len(filter.Location) == 0 {
		return nil, constants.ErrorRequiredLocation
	}

	return processAMIList(filter.Location, filter.Tags)
}
