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
	"time"

	"emperror.dev/emperror"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/config"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// EKSProfile describes an Amazon EKS cluster profile
type EKSProfile struct {
	DefaultModel
	Region     string                `gorm:"default:'us-west-2'"`
	Version    string                `gorm:"default:'1.10'"`
	NodePools  []*EKSNodePoolProfile `gorm:"foreignkey:Name"`
	TtlMinutes uint                  `gorm:"not null;default:0"`
}

// EKSNodePoolProfile describes an EKS cluster profile's nodepools
type EKSNodePoolProfile struct {
	AmazonNodePoolProfileBaseFields
	Image  string                      `gorm:"default:'ami-0a54c984b9f908c81'"`
	Labels []*EKSNodePoolLabelsProfile `gorm:"foreignkey:NodePoolProfileID"`
}

// EKSNodePoolLabelsProfile describe the labels of a nodepool
// of an EKS cluster profile
type EKSNodePoolLabelsProfile struct {
	ID                uint   `gorm:"primary_key"`
	Name              string `gorm:"unique_index:idx_eks_profile_node_pool_labels_id_name"`
	Value             string
	NodePoolProfileID uint `gorm:"unique_index:idx_eks_profile_node_pool_labels_id_name"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TableName overrides EKSProfile's table name
func (EKSProfile) TableName() string {
	return DefaultEKSProfileTableName
}

// TableName overrides EKSNodePoolProfile's table name
func (EKSNodePoolProfile) TableName() string {
	return DefaultEKSNodePoolProfileTableName
}

// TableName override the EKSNodePoolLabelsProfile's table name
func (EKSNodePoolLabelsProfile) TableName() string {
	return DefaultEKSNodePoolLabelsProfileTableName
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

			labels := make(map[string]string)
			for _, lbl := range np.Labels {
				if lbl != nil {
					labels[lbl.Name] = lbl.Value
				}
			}

			nodePools[np.NodeName] = &eks.NodePool{
				InstanceType: np.InstanceType,
				SpotPrice:    np.SpotPrice,
				Autoscaling:  np.Autoscaling,
				MinCount:     np.MinCount,
				MaxCount:     np.MaxCount,
				Count:        np.Count,
				Image:        np.Image,
				Labels:       labels,
			}
		}
	}

	return &pkgCluster.ClusterProfileResponse{
		Name:       d.DefaultModel.Name,
		Location:   d.Region,
		Cloud:      pkgCluster.Amazon,
		TtlMinutes: d.TtlMinutes,
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

	image, err := eks.GetDefaultImageID(d.Region, d.Version) // the image is fixed for a region
	if err != nil {
		return emperror.Wrapf(err, "couldn't get EKS AMI for Kubernetes version %q in region %q", d.Version, d.Region)
	}

	if r.Properties.EKS == nil || len(r.Properties.EKS.NodePools) == 0 {
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

				err := pkgCommon.ValidateNodePoolLabels(nodePool.Labels)
				if err != nil {
					return err
				}

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

				var labels []*EKSNodePoolLabelsProfile

				for lblName, lblValue := range nodePool.Labels {
					labels = append(labels, &EKSNodePoolLabelsProfile{
						Name:  lblName,
						Value: lblValue,
					})
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
					Image:  image,
					Labels: labels,
				})

			}

			d.NodePools = nodePools
		}
	}

	d.TtlMinutes = r.TtlMinutes

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

// BeforeSave clears nodepools
func (d *EKSProfile) BeforeUpdate(tx *gorm.DB) error {
	log.Info("BeforeUpdate eks profile...")

	if d.CreatedAt.IsZero() && d.UpdatedAt.IsZero() {
		return tx.Create(d).Error
	}

	var nodePools []*EKSNodePoolProfile

	err := tx.Where(EKSNodePoolProfile{
		AmazonNodePoolProfileBaseFields: AmazonNodePoolProfileBaseFields{
			Name: d.Name,
		},
	}).Find(&nodePools).Delete(&nodePools).Error
	if err != nil {
		return err
	}

	return nil
}

// BeforeDelete deletes all nodepools to belongs to profile
func (d *EKSProfile) BeforeDelete(tx *gorm.DB) error {
	log.Info("BeforeDelete eks profile... delete all nodepool")

	var nodePools []*EKSNodePoolProfile

	err := tx.Where(EKSNodePoolProfile{
		AmazonNodePoolProfileBaseFields: AmazonNodePoolProfileBaseFields{
			Name: d.Name,
		},
	}).Find(&nodePools).Delete(&nodePools).Error
	if err != nil {
		return err
	}

	return nil
}

// BeforeDelete deletes all node labels that belong to node pool profile
func (d *EKSNodePoolProfile) BeforeDelete(tx *gorm.DB) error {
	log.Info("BeforeDelete eks profile... delete all nodepool")

	if d.ID == 0 {
		return nil
	}

	err := tx.Where(EKSNodePoolLabelsProfile{
		NodePoolProfileID: d.ID,
	}).Delete(EKSNodePoolLabelsProfile{}).Error
	if err != nil {
		return err
	}

	return nil
}
