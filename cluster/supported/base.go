package supported

import (
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/go-errors/errors"
)

var logger *logrus.Logger
var log *logrus.Entry

// todo move to BT
const (
	Location          = "location"
	InstanceType      = "instanceType"
	KubernetesVersion = "k8sVersion"
)

var (
	Keywords = []string{
		Location,
		InstanceType,
		KubernetesVersion,
	}
)

// Simple init for logging
func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"tag": "Supported"})
}

type CloudInfoProvider interface {
	GetType() string
	GetNameRegexp() string
	GetLocations() ([]string, error)
	GetMachineTypes() (map[string]cluster.MachineType, error)
	GetMachineTypesWithFilter(*InstanceFilter) (map[string]cluster.MachineType, error)
	GetKubernetesVersion(*KubernetesFilter) (interface{}, error)
}

type BaseFields struct {
	OrgId    uint
	SecretId string
}

// todo move to BT
type CloudInfoRequest struct {
	OrganizationId uint   `json:"-"`
	SecretId       string `json:"secret_id,omitempty"`
	Filter *struct {
		Fields           []string          `json:"fields,omitempty"`
		InstanceType     *InstanceFilter   `json:"instanceType,omitempty"`
		KubernetesFilter *KubernetesFilter `json:"k8sVersion,omitempty"`
	} `json:"filter,omitempty"`
	Google *struct {
		ProjectId string `json:"project_id,omitempty"` // todo secret?
	} `json:"google,omitempty"`
}

type InstanceFilter struct {
	Zone string    `json:"zone,omitempty"`
	Tags []*string `json:"tags,omitempty"`
}

type KubernetesFilter struct {
	Zone string `json:"zone,omitempty"`
}

// todo move to BT
type GetCloudInfoResponse struct {
	Type               string                         `json:"type" binding:"required"`
	NameRegexp         string                         `json:"nameRegexp,omitempty"`
	Locations          []string                       `json:"locations,omitempty"`
	NodeInstanceType   map[string]cluster.MachineType `json:"nodeInstanceType,omitempty"`
	KubernetesVersions interface{}                    `json:"kubernetes_versions,omitempty"`
}

func GetCloudInfoModel(cloudType string, r *CloudInfoRequest) (CloudInfoProvider, error) {
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
		var projectId string
		if r.Google != nil {
			projectId = r.Google.ProjectId
		}
		return &GoogleInfo{
			BaseFields: BaseFields{
				OrgId:    r.OrganizationId,
				SecretId: r.SecretId,
			},
			ProjectId: projectId,
		}, nil
	case constants.Azure:
		if len(r.SecretId) != 0 {
			return &AzureInfo{
				BaseFields: BaseFields{
					OrgId:    r.OrganizationId,
					SecretId: r.SecretId,
				},
			}, nil
		} else {
			return nil, errors.New("Secret id is required") // todo move to BT
		}
	default:
		return nil, constants.ErrorNotSupportedCloudType
	}
}

func ProcessFilter(p CloudInfoProvider, r *CloudInfoRequest) (*GetCloudInfoResponse, error) {

	response := GetCloudInfoResponse{
		Type:       p.GetType(),
		NameRegexp: p.GetNameRegexp(),
	}
	if r != nil && r.Filter != nil {
		for _, field := range r.Filter.Fields {
			switch field {

			case Location:
				if l, err := p.GetLocations(); err != nil {
					return nil, err
				} else {
					response.Locations = l
				}

			case InstanceType:
				if r.Filter.InstanceType != nil {
					log.Infof("Get machine types with filter [%s]", &r.Filter.InstanceType)
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

			case KubernetesVersion:
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
