package supported

import (
	"github.com/banzaicloud/pipeline/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// AmazonInfo describes AWS with supported info
type AmazonInfo struct {
	BaseFields
}

const defaultRegion = "eu-west-1"

// GetType returns cloud type
func (a *AmazonInfo) GetType() string {
	return pkgCluster.Amazon
}

// GetNameRegexp returns regexp for cluster name
func (a *AmazonInfo) GetNameRegexp() string {
	return pkgCluster.RegexpAWSName
}

// GetLocations returns supported locations
func (a *AmazonInfo) GetLocations() ([]string, error) {

	if len(a.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}

	regions, err := cluster.ListRegions(a.OrgId, a.SecretId, defaultRegion)
	if err != nil {
		return nil, err
	}

	var locations []string
	for _, region := range regions {
		locations = append(locations, *region.RegionName)
	}
	return locations, nil
}

// GetMachineTypes returns supported machine types
func (a *AmazonInfo) GetMachineTypes() (map[string]pkgCluster.MachineType, error) {
	return nil, pkgErrors.ErrorRequiredLocation
}

// GetMachineTypesWithFilter returns supported machine types by location
func (a *AmazonInfo) GetMachineTypesWithFilter(filter *pkgCluster.InstanceFilter) (map[string]pkgCluster.MachineType, error) {
	// todo NOTE: until aws sdk dont support getting instance types
	response := make(map[string]pkgCluster.MachineType)
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
func (a *AmazonInfo) GetKubernetesVersion(*pkgCluster.KubernetesFilter) (interface{}, error) {
	return nil, pkgErrors.ErrorCloudInfoK8SNotSupported
}

// processAMIList returns supported AMIs by region and tags
func (a *AmazonInfo) processAMIList(region string, tags []*string) (map[string][]string, error) {
	amiList, err := cluster.ListAMIs(a.OrgId, a.SecretId, region, tags)
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
func (a *AmazonInfo) GetImages(filter *pkgCluster.ImageFilter) (map[string][]string, error) {

	if len(a.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}

	if len(filter.Location) == 0 {
		return nil, pkgErrors.ErrorRequiredLocation
	}

	return a.processAMIList(filter.Location, filter.Tags)
}
