package supported

import (
	"github.com/banzaicloud/pipeline/config"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Simple init for logging
func init() {
	log = config.Logger()
}

// CloudInfoProvider interface for cloud supports
type CloudInfoProvider interface {
	GetType() string
	GetNameRegexp() string
	GetLocations() ([]string, error)
	GetMachineTypes() (map[string]pkgCluster.MachineType, error)
	GetMachineTypesWithFilter(*pkgCluster.InstanceFilter) (map[string]pkgCluster.MachineType, error)
	GetKubernetesVersion(*pkgCluster.KubernetesFilter) (interface{}, error)
	GetImages(*pkgCluster.ImageFilter) (map[string][]string, error)
}

// BaseFields for cloud info types
type BaseFields struct {
	OrgId    uint
	SecretId string
}

// GetCloudInfoModel creates CloudInfoProvider
func GetCloudInfoModel(cloudType string, r *pkgCluster.CloudInfoRequest) (CloudInfoProvider, error) {
	log.Infof("Cloud type: %s", cloudType)
	switch cloudType {

	case pkgCluster.Amazon:
		return &AmazonInfo{
			BaseFields: BaseFields{
				OrgId:    r.OrganizationId,
				SecretId: r.SecretId,
			},
		}, nil

	case pkgCluster.Google:
		return &GoogleInfo{
			BaseFields: BaseFields{
				OrgId:    r.OrganizationId,
				SecretId: r.SecretId,
			},
		}, nil

	case pkgCluster.Azure:
		return &AzureInfo{
			BaseFields: BaseFields{
				OrgId:    r.OrganizationId,
				SecretId: r.SecretId,
			},
		}, nil

	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}
}

// ProcessFilter returns the proper supported fields, the CloudInfoRequest decide which
func ProcessFilter(p CloudInfoProvider, r *pkgCluster.CloudInfoRequest) (*pkgCluster.GetCloudInfoResponse, error) {

	response := pkgCluster.GetCloudInfoResponse{
		Type:       p.GetType(),
		NameRegexp: p.GetNameRegexp(),
	}
	if r != nil && r.Filter != nil {
		for _, field := range r.Filter.Fields {
			switch field {

			case pkgCluster.KeyWordLocation:
				l, err := p.GetLocations()
				if err != nil {
					return nil, err
				}
				response.Locations = l

			case pkgCluster.KeyWordInstanceType:
				if r.Filter.InstanceType != nil {
					log.Infof("Get machine types with filter [%#v]", *r.Filter.InstanceType)
					// get machine types from spec zone
					mt, err := p.GetMachineTypesWithFilter(r.Filter.InstanceType)
					if err != nil {
						return nil, err
					}
					response.NodeInstanceType = mt
				} else {
					// get machine types from all zone
					log.Info("Get machine types from all zone")
					mt, err := p.GetMachineTypes()
					if err != nil {
						return nil, err
					}
					response.NodeInstanceType = mt
				}

			case pkgCluster.KeyWordKubernetesVersion:
				versions, err := p.GetKubernetesVersion(r.Filter.KubernetesFilter)
				if err != nil {
					return nil, err
				}
				response.KubernetesVersions = versions
			case pkgCluster.KeyWordImage:
				images, err := p.GetImages(r.Filter.ImageFilter)
				if err != nil {
					return nil, err
				}
				response.Image = images
			}
		}
	} else {
		log.Info("Filter field is empty")
	}

	return &response, nil

}
