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
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
)

// Provider name regexp
const (
	RegexpOKEName = `^[A-z0-9-_]{1,255}$`
)

// OracleInfo describes OKE with supported info
type OracleInfo struct {
	BaseFields
}

// GetOCI gets an initialized oci.OCI
func (oi *OracleInfo) GetOCI(orgId uint, secretId string) (OCI *oci.OCI, err error) {

	oc, err := cluster.CreateOKEClusterFromModel(&model.ClusterModel{
		OrganizationId: orgId,
		SecretId:       secretId,
		Cloud:          pkgCluster.Oracle,
	})
	if err != nil {
		return OCI, err
	}

	return oc.GetOCI()
}

// GetType returns cloud type
func (oi *OracleInfo) GetType() string {
	return pkgCluster.Oracle
}

// GetNameRegexp returns regexp for cluster name
func (oi *OracleInfo) GetNameRegexp() string {
	return RegexpOKEName
}

// GetLocations returns supported locations
func (oi *OracleInfo) GetLocations() ([]string, error) {
	if len(oi.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}

	oci, err := oi.GetOCI(oi.BaseFields.OrgId, oi.BaseFields.SecretId)
	if err != nil {
		return nil, err
	}

	ic, err := oci.NewIdentityClient()
	if err != nil {
		return nil, err
	}

	regions, err := ic.GetSubscribedRegionNames()
	if err != nil {
		return nil, err
	}

	_regions := make([]string, 0)
	for _, region := range regions {
		_regions = append(_regions, region)
	}

	return _regions, nil
}

// GetMachineTypes returns supported machine types
func (oi *OracleInfo) GetMachineTypes() (map[string]pkgCluster.MachineType, error) {
	if len(oi.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}

	oci, err := oi.GetOCI(oi.BaseFields.OrgId, oi.BaseFields.SecretId)
	if err != nil {
		return nil, err
	}

	shapesByRegion, err := oci.GetSupportedShapes()
	if err != nil {
		return nil, err
	}

	_shapes := make(map[string]pkgCluster.MachineType, 0)
	for region, shapes := range shapesByRegion {
		_shapes[region] = shapes
	}

	return _shapes, nil
}

// GetMachineTypesWithFilter returns supported machine types by location
func (oi *OracleInfo) GetMachineTypesWithFilter(filter *pkgCluster.InstanceFilter) (map[string]pkgCluster.MachineType, error) {

	if len(oi.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}

	if len(filter.Location) == 0 {
		return nil, pkgErrors.ErrorRequiredLocation
	}

	oci, err := oi.GetOCI(oi.BaseFields.OrgId, oi.BaseFields.SecretId)
	if err != nil {
		return nil, err
	}

	shapes, err := oci.GetSupportedShapesInARegion(filter.Location)
	if err != nil {
		return nil, err
	}

	_shapes := make(map[string]pkgCluster.MachineType, 0)
	_shapes[filter.Location] = shapes

	return _shapes, nil
}

// GetKubernetesVersion returns supported k8s versions
func (oi *OracleInfo) GetKubernetesVersion(filter *pkgCluster.KubernetesFilter) (interface{}, error) {

	if len(oi.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}

	oci, err := oi.GetOCI(oi.BaseFields.OrgId, oi.BaseFields.SecretId)
	if err != nil {
		return nil, err
	}

	if filter == nil || len(filter.Location) == 0 {
		return oci.GetSupportedK8SVersions()
	}

	versionsByRegion := make(map[string][]string, 0)
	versions, err := oci.GetSupportedK8SVersionsInARegion(filter.Location)
	if err != nil {
		return nil, err
	}

	versionsByRegion[filter.Location] = versions

	return versionsByRegion, nil
}

// GetImages returns with the supported images
func (oi *OracleInfo) GetImages(filter *pkgCluster.ImageFilter) (map[string][]string, error) {
	if len(oi.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}

	oci, err := oi.GetOCI(oi.BaseFields.OrgId, oi.BaseFields.SecretId)
	if err != nil {
		return nil, err
	}

	if filter == nil || len(filter.Location) == 0 {
		return oci.GetSupportedImages()
	}
	imagesByRegion := make(map[string][]string, 0)
	images, err := oci.GetSupportedImagesInARegion(filter.Location)
	if err != nil {
		return nil, err
	}

	imagesByRegion[filter.Location] = images

	return imagesByRegion, nil
}
