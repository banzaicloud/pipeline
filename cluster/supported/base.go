package supported

import (
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/components"
)

var logger *logrus.Logger
var log *logrus.Entry

var (
	Keywords = []string{
		constants.KeyWorldLocation,
		constants.KeyWorldInstanceType,
		constants.KeyWorldKubernetesVersion,
	}
)

// Simple init for logging
func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"tag": "Supported"})
}

// CloudInfoProvider interface for cloud supports
type CloudInfoProvider interface {
	GetType() string
	GetNameRegexp() string
	GetLocations() ([]string, error)
	GetMachineTypes() (map[string]components.MachineType, error)
	GetMachineTypesWithFilter(*components.InstanceFilter) (map[string]components.MachineType, error)
	GetKubernetesVersion(*components.KubernetesFilter) (interface{}, error)
}

// Base fields for cloud info types
type BaseFields struct {
	OrgId    uint
	SecretId string
}

// GetCloudInfoModel creates CloudInfoProvider
func GetCloudInfoModel(cloudType string, r *components.CloudInfoRequest) (CloudInfoProvider, error) {
	log.Infof("Cloud type: %s", cloudType)
	switch cloudType {

	case constants.Amazon:
		return &AmazonInfo{
			BaseFields: BaseFields{
				OrgId:    r.OrganizationId,
				SecretId: r.SecretId,
			},
		}, nil

	case constants.Google:
		return &GoogleInfo{
			BaseFields: BaseFields{
				OrgId:    r.OrganizationId,
				SecretId: r.SecretId,
			},
		}, nil

	case constants.Azure:
		return &AzureInfo{
			BaseFields: BaseFields{
				OrgId:    r.OrganizationId,
				SecretId: r.SecretId,
			},
		}, nil

	default:
		return nil, constants.ErrorNotSupportedCloudType
	}
}

func ProcessFilter(p CloudInfoProvider, r *components.CloudInfoRequest) (*components.GetCloudInfoResponse, error) {

	response := components.GetCloudInfoResponse{
		Type:       p.GetType(),
		NameRegexp: p.GetNameRegexp(),
	}
	if r != nil && r.Filter != nil {
		for _, field := range r.Filter.Fields {
			switch field {

			case constants.KeyWorldLocation:
				if l, err := p.GetLocations(); err != nil {
					return nil, err
				} else {
					response.Locations = l
				}

			case constants.KeyWorldInstanceType:
				if r.Filter.InstanceType != nil {
					log.Infof("Get machine types with filter [%#v]", *r.Filter.InstanceType)
					// get machine types from spec zone
					if mt, err := p.GetMachineTypesWithFilter(r.Filter.InstanceType); err != nil {
						return nil, err
					} else {
						response.NodeInstanceType = mt
					}
				} else {
					// get machine types from all zone
					log.Info("Get machine types from all zone")
					if mt, err := p.GetMachineTypes(); err != nil {
						return nil, err
					} else {
						response.NodeInstanceType = mt
					}
				}

			case constants.KeyWorldKubernetesVersion:
				if versions, err := p.GetKubernetesVersion(r.Filter.KubernetesFilter); err != nil {
					return nil, err
				} else {
					response.KubernetesVersions = versions
				}

			}
		}
	} else {
		log.Info("Filter field is empty")
	}

	return &response, nil

}
