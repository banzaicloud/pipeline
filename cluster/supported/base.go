// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package supported

import (
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

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

	case pkgCluster.Oracle:
		return &OracleInfo{
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
				if r.Filter.InstanceType != nil && r.Filter.InstanceType.Location != "" {
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
