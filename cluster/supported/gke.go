package supported

import (
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/go-errors/errors"
)

type GoogleInfo struct {
	BaseFields
	ProjectId string
	Zone      string
}

func (g *GoogleInfo) GetType() string {
	return constants.Google
}
func (g *GoogleInfo) GetNameRegexp() string {
	return constants.RegexpGKEName
}

func (g *GoogleInfo) GetLocations() ([]string, error) {
	if len(g.ProjectId) == 0 {
		return nil, errors.New("Project id is required") // todo move to BT
	}
	return cluster.GetZones(g.ProjectId)
}

func (g *GoogleInfo) GetMachineTypes() (map[string]cluster.MachineType, error) {
	if len(g.ProjectId) == 0 {
		return nil, errors.New("Project id is required") // todo move to BT
	}
	return cluster.GetAllMachineTypes(g.ProjectId)
}

func (g *GoogleInfo) GetMachineTypesWithFilter(filter *InstanceFilter) (map[string]cluster.MachineType, error) {

	if len(g.ProjectId) == 0 {
		return nil, errors.New("Project id is required") // todo move to BT
	}

	if len(filter.Zone) == 0 {
		return nil, errors.New("SubField is required") // todo move to BT
	}

	return cluster.GetAllMachineTypesByZone(g.ProjectId, filter.Zone)
}

func (g *GoogleInfo) GetKubernetesVersion() (interface{}, error) {

	if len(g.SecretId) == 0 {
		return nil, errors.New("Secret id is required")
	}

	if len(g.Zone) == 0 {
		return nil, errors.New("Zone is required")
	}

	return cluster.GetGkeServerConfig(g.OrgId, g.SecretId, g.ProjectId, g.Zone)
}
