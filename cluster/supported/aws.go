package supported

import (
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/go-errors/errors"
)

type AmazonInfo struct {
	BaseFields
}

func (a *AmazonInfo) GetType() string {
	return constants.Amazon
}

func (a *AmazonInfo) GetNameRegexp() string {
	return constants.RegexpAWSName
}

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

func (a *AmazonInfo) GetMachineTypes() (map[string]cluster.MachineType, error) {
	return nil, errors.New("Zone is required") // todo move to BT
}

func (a *AmazonInfo) GetMachineTypesWithFilter(filter *InstanceFilter) (map[string]cluster.MachineType, error) {

	if len(filter.Zone) == 0 {
		return nil, errors.New("Zone is required") // todo move to BT
	}

	return processMachineTypes(filter.Zone, filter.Tags)
}

func (a *AmazonInfo) GetKubernetesVersion() (interface{}, error) {
	return nil, nil
}

func processMachineTypes(region string, tags []*string) (map[string]cluster.MachineType, error) {
	amiList, err := cluster.ListAMIs(region, tags)
	if err != nil {
		return nil, err
	}

	response := make(map[string]cluster.MachineType)
	var amiSlice []string
	for _, ami := range amiList {
		amiSlice = append(amiSlice, *ami.ImageId)
	}

	response[region] = amiSlice

	return response, nil
}
