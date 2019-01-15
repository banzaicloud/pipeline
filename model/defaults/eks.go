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

package defaults

import (
	"github.com/banzaicloud/pipeline/config"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// EKSProfile describes an Amazon EKS cluster profile
type EKSProfile struct {
	DefaultModel
	Region    string                `gorm:"default:'us-west-2'"`
	Version   string                `gorm:"default:'1.10'"`
	NodePools []*EKSNodePoolProfile `gorm:"foreignkey:Name"`
}

// EKSNodePoolProfile describes an EKS cluster profile's nodepools
type EKSNodePoolProfile struct {
	AmazonNodePoolProfileBaseFields
	Image string `gorm:"default:'ami-0a54c984b9f908c81'"`
}

// TableName overrides EKSProfile's table name
func (EKSProfile) TableName() string {
	return DefaultEKSProfileTableName
}

// TableName overrides EKSNodePoolProfile's table name
func (EKSNodePoolProfile) TableName() string {
	return DefaultEKSNodePoolProfileTableName
}

// SaveInstance saves cluster profile into database
func (d *EKSProfile) SaveInstance() error {
	return save(d)
}

// GetCloud returns profile's cloud type
func (d *EKSProfile) GetCloud() string {
	return pkgCluster.Amazon
}

// GetDistribution returns profile's distribution type
func (d *EKSProfile) GetDistribution() string {
	return pkgCluster.EKS
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *EKSProfile) IsDefinedBefore() bool {
	return config.DB().First(&d).RowsAffected != 0
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *EKSProfile) GetProfile() *pkgCluster.ClusterProfileResponse {

	nodePools := make(map[string]*eks.NodePool)
	for _, np := range d.NodePools {
		if np != nil {
			nodePools[np.NodeName] = &eks.NodePool{
				InstanceType: np.InstanceType,
				SpotPrice:    np.SpotPrice,
				Autoscaling:  np.Autoscaling,
				MinCount:     np.MinCount,
				MaxCount:     np.MaxCount,
				Count:        np.Count,
				Image:        np.Image,
			}
		}
	}

	return &pkgCluster.ClusterProfileResponse{
		Name:     d.DefaultModel.Name,
		Location: d.Region,
		Cloud:    pkgCluster.Amazon,
		Properties: &pkgCluster.ClusterProfileProperties{
			EKS: &eks.ClusterProfileEKS{
				Version:   d.Version,
				NodePools: nodePools,
			},
		},
	}

}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *EKSProfile) UpdateProfile(r *pkgCluster.ClusterProfileRequest, withSave bool) error {

	regionChanged := false
	if len(r.Location) != 0 && r.Location != d.Region {
		d.Region = r.Location
		regionChanged = true
	}
	image := eks.DefaultImages[d.Version][d.Region] // the image is fixed for a region

	if regionChanged && (r.Properties.EKS == nil || len(r.Properties.EKS.NodePools) == 0) {
		for _, np := range d.NodePools {
			np.Image = image
		}
	}

	if r.Properties.EKS != nil {

		if len(r.Properties.EKS.Version) != 0 {
			d.Version = r.Properties.EKS.Version
		}

		if len(r.Properties.EKS.NodePools) != 0 {
			var nodePools []*EKSNodePoolProfile
			for npName, nodePool := range r.Properties.EKS.NodePools {

				spotPrice := eks.DefaultSpotPrice
				instanceType := eks.DefaultInstanceType
				minCount := pkgCommon.DefaultNodeMinCount
				maxCount := pkgCommon.DefaultNodeMaxCount

				if len(nodePool.SpotPrice) != 0 {
					spotPrice = nodePool.SpotPrice
				}

				if len(nodePool.InstanceType) != 0 {
					instanceType = nodePool.InstanceType
				}

				if nodePool.MinCount != 0 {
					minCount = nodePool.MinCount
				}

				if nodePool.MaxCount != 0 {
					maxCount = nodePool.MaxCount
				}

				if minCount > maxCount {
					minCount = pkgCommon.DefaultNodeMinCount
					maxCount = pkgCommon.DefaultNodeMaxCount
				}

				count := nodePool.Count
				if count == 0 {
					count = minCount
				}

				nodePools = append(nodePools, &EKSNodePoolProfile{
					AmazonNodePoolProfileBaseFields: AmazonNodePoolProfileBaseFields{
						InstanceType: instanceType,
						Name:         d.Name,
						NodeName:     npName,
						SpotPrice:    spotPrice,
						Autoscaling:  nodePool.Autoscaling,
						MinCount:     minCount,
						MaxCount:     maxCount,
						Count:        count,
					},
					Image: image,
				})

			}

			d.NodePools = nodePools
		}
	}
	if withSave {
		return d.SaveInstance()
	}
	d.Name = r.Name
	return nil
}

// DeleteProfile deletes cluster profile from database
func (d *EKSProfile) DeleteProfile() error {
	return config.DB().Delete(&d).Error
}

// AfterFind loads nodepools to profile
func (d *EKSProfile) AfterFind() error {
	log.Info("AfterFind eks profile... load node pools")
	return config.DB().Where(EKSNodePoolProfile{
		AmazonNodePoolProfileBaseFields: AmazonNodePoolProfileBaseFields{
			Name: d.Name,
		},
	}).Find(&d.NodePools).Error
}

// BeforeSave clears nodepools
func (d *EKSProfile) BeforeSave() error {
	log.Info("BeforeSave eks profile...")

	db := config.DB()
	var nodePools []*EKSNodePoolProfile
	err := db.Where(EKSNodePoolProfile{
		AmazonNodePoolProfileBaseFields: AmazonNodePoolProfileBaseFields{
			Name: d.Name,
		},
	}).Find(&nodePools).Delete(&nodePools).Error
	if err != nil {
		log.Errorf("Error during deleting saved nodepools: %s", err.Error())
	}

	return nil
}

// BeforeDelete deletes all nodepools to belongs to profile
func (d *EKSProfile) BeforeDelete() error {
	log.Info("BeforeDelete eks profile... delete all nodepool")

	var nodePools []*EKSNodePoolProfile
	return config.DB().Where(EKSNodePoolProfile{
		AmazonNodePoolProfileBaseFields: AmazonNodePoolProfileBaseFields{
			Name: d.Name,
		},
	}).Find(&nodePools).Delete(&nodePools).Error
}
