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

	"github.com/banzaicloud/pipeline/config"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/aks"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/jinzhu/gorm"
)

// AKSProfile describes an Azure cluster profile
type AKSProfile struct {
	DefaultModel
	Location          string                `gorm:"default:'eastus'"`
	KubernetesVersion string                `gorm:"default:'1.9.2'"`
	NodePools         []*AKSNodePoolProfile `gorm:"foreignkey:Name"`
}

// AKSNodePoolProfile describes an Azure cluster profile's nodepools
type AKSNodePoolProfile struct {
	ID               uint                        `gorm:"primary_key"`
	Autoscaling      bool                        `gorm:"default:false"`
	MinCount         int                         `gorm:"default:1"`
	MaxCount         int                         `gorm:"default:2"`
	Count            int                         `gorm:"default:1"`
	NodeInstanceType string                      `gorm:"default:'Standard_D4_v2'"`
	Name             string                      `gorm:"unique_index:idx_name_node_name"`
	NodeName         string                      `gorm:"unique_index:idx_name_node_name"`
	Labels           []*AKSNodePoolLabelsProfile `gorm:"foreignkey:NodePoolProfileID"`
}

// AKSNodePoolLabelsProfile stores labels for Azure cluster profile's nodepools
type AKSNodePoolLabelsProfile struct {
	ID                uint   `gorm:"primary_key"`
	Name              string `gorm:"unique_index:idx_name_profile_node_pool_id"`
	Value             string
	NodePoolProfileID uint `gorm:"unique_index:idx_name_profile_node_pool_id"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TableName overrides AKSNodePoolProfile's table name
func (AKSNodePoolProfile) TableName() string {
	return DefaultAKSNodePoolProfileTableName
}

// TableName overrides AKSProfile's table name
func (AKSProfile) TableName() string {
	return DefaultAKSProfileTableName
}

// TableName overrides AKSProfile's table name
func (AKSNodePoolLabelsProfile) TableName() string {
	return DefaultAKSNodePoolProfileLabelsTableName
}

// AfterFind loads nodepools to profile
func (d *AKSProfile) AfterFind() error {
	log.Info("AfterFind aks profile... load node pools")
	return config.DB().Where(AKSNodePoolProfile{Name: d.Name}).Find(&d.NodePools).Error
}

// BeforeUpdate clears nodepools
func (d *AKSProfile) BeforeUpdate(tx *gorm.DB) error {
	log.Info("BeforeUpdate aks profile...")

	if d.CreatedAt.IsZero() && d.UpdatedAt.IsZero() {
		return tx.Create(d).Error
	}

	var nodePools []*AKSNodePoolProfile
	err := tx.Where(AKSNodePoolProfile{
		Name: d.Name,
	}).Find(&nodePools).Delete(&nodePools).Error
	if err != nil {
		return err
	}

	return nil
}

// BeforeDelete deletes all nodepools to belongs to profile
func (d *AKSProfile) BeforeDelete(tx *gorm.DB) error {
	log.Info("BeforeDelete aks profile... delete all nodepool")

	var nodePools []*AKSNodePoolProfile

	err := tx.Where(AKSNodePoolProfile{
		Name: d.Name,
	}).Find(&nodePools).Delete(&nodePools).Error
	if err != nil {
		return err
	}

	return nil
}

// BeforeDelete deletes all labels belongs to the nodepool
func (d *AKSNodePoolProfile) BeforeDelete(tx *gorm.DB) error {
	log.Info("BeforeDelete aks profile... delete all nodepool")

	if d.ID == 0 {
		return nil
	}

	err := tx.Where(AKSNodePoolLabelsProfile{
		NodePoolProfileID: d.ID,
	}).Delete(AKSNodePoolLabelsProfile{}).Error
	if err != nil {
		return err
	}

	return nil
}

// SaveInstance saves cluster profile into database
func (d *AKSProfile) SaveInstance() error {
	return save(d)
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *AKSProfile) IsDefinedBefore() bool {
	return config.DB().First(&d).RowsAffected != int64(0)
}

// GetCloud returns profile's cloud type
func (d *AKSProfile) GetCloud() string {
	return pkgCluster.Azure
}

// GetDistribution returns profile's distribution type
func (d *AKSProfile) GetDistribution() pkgCluster.DistributionID {
	return pkgCluster.AKS
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *AKSProfile) GetProfile() *pkgCluster.ClusterProfileResponse {

	nodePools := make(map[string]*aks.NodePoolCreate)
	for _, np := range d.NodePools {
		if np != nil {

			labels := make(map[string]string)
			for _, lbl := range np.Labels {
				if lbl != nil {
					labels[lbl.Name] = lbl.Value
				}
			}

			nodePools[np.NodeName] = &aks.NodePoolCreate{
				Autoscaling:      np.Autoscaling,
				MinCount:         np.MinCount,
				MaxCount:         np.MaxCount,
				Count:            np.Count,
				NodeInstanceType: np.NodeInstanceType,
				Labels:           labels,
			}
		}
	}

	return &pkgCluster.ClusterProfileResponse{
		Name:     d.DefaultModel.Name,
		Location: d.Location,
		Cloud:    pkgCluster.Azure,
		Properties: &pkgCluster.ClusterProfileProperties{
			AKS: &aks.ClusterProfileAKS{
				KubernetesVersion: d.KubernetesVersion,
				NodePools:         nodePools,
			},
		},
	}
}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *AKSProfile) UpdateProfile(r *pkgCluster.ClusterProfileRequest, withSave bool) error {
	if len(r.Location) != 0 {
		d.Location = r.Location
	}

	if r.Properties.AKS != nil {

		if len(r.Properties.AKS.KubernetesVersion) != 0 {
			d.KubernetesVersion = r.Properties.AKS.KubernetesVersion
		}

		if len(r.Properties.AKS.NodePools) != 0 {

			var nodePools []*AKSNodePoolProfile
			for name, np := range r.Properties.AKS.NodePools {

				err := pkgCommon.ValidateNodePoolLabels(np.Labels)
				if err != nil {
					return err
				}

				labels := make([]*AKSNodePoolLabelsProfile, 0, len(np.Labels))
				for name, value := range np.Labels {
					labels = append(labels, &AKSNodePoolLabelsProfile{
						Name:  name,
						Value: value,
					})
				}
				nodePools = append(nodePools, &AKSNodePoolProfile{
					Autoscaling:      np.Autoscaling,
					MinCount:         np.MinCount,
					MaxCount:         np.MaxCount,
					Count:            np.Count,
					NodeInstanceType: np.NodeInstanceType,
					Name:             d.Name,
					NodeName:         name,
					Labels:           labels,
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
func (d *AKSProfile) DeleteProfile() error {
	return config.DB().Delete(&d).Error
}
