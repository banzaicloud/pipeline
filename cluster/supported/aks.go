package supported

import (
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/banzai-types/constants"
	"errors"
)

type AzureInfo struct {
	BaseFields
}

func (a *AzureInfo) GetType() string {
	return constants.Azure
}

func (a *AzureInfo) GetNameRegexp() string {
	return constants.RegexpAKSName
}

func (a *AzureInfo) GetLocations() ([]string, error) {
	return cluster.GetLocations(a.OrgId, a.SecretId)
}

func (a *AzureInfo) GetMachineTypes() (map[string]cluster.MachineType, error) {
	return nil, errors.New("Zone is required") // todo move to BT
}

func (a *AzureInfo) GetMachineTypesWithFilter(filter *InstanceFilter) (map[string]cluster.MachineType, error) {

	if len(filter.Zone) == 0 {
		return nil, errors.New("Zone is required") // todo move to BT
	}

	return cluster.GetMachineTypes(a.OrgId, a.SecretId, filter.Zone)
}

func (a *AzureInfo) GetKubernetesVersion(filter *KubernetesFilter) (interface{}, error) {

	if filter == nil || len(filter.Zone) == 0 {
		return nil, errors.New("Zone is required") // todo move to BT
	}

	return cluster.GetKubernetesVersion(a.OrgId, a.SecretId, filter.Zone)
}
