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
	"github.com/banzaicloud/pipeline/pkg/cluster/aks"
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
	ID               uint   `gorm:"primary_key"`
	Autoscaling      bool   `gorm:"default:false"`
	MinCount         int    `gorm:"default:1"`
	MaxCount         int    `gorm:"default:2"`
	Count            int    `gorm:"default:1"`
	NodeInstanceType string `gorm:"default:'Standard_D4_v2'"`
	Name             string `gorm:"unique_index:idx_name_node_name"`
	NodeName         string `gorm:"unique_index:idx_name_node_name"`
}

// TableName overrides AKSNodePoolProfile's table name
func (AKSNodePoolProfile) TableName() string {
	return DefaultAKSNodePoolProfileTableName
}

// TableName overrides AKSProfile's table name
func (AKSProfile) TableName() string {
	return DefaultAKSProfileTableName
}

// AfterFind loads nodepools to profile
func (d *AKSProfile) AfterFind() error {
	log.Info("AfterFind aks profile... load node pools")
	return config.DB().Where(AKSNodePoolProfile{Name: d.Name}).Find(&d.NodePools).Error
}

// BeforeSave clears nodepools
func (d *AKSProfile) BeforeSave() error {
	log.Info("BeforeSave aks profile...")

	db := config.DB()
	var nodePools []*AKSNodePoolProfile
	err := db.Where(AKSNodePoolProfile{
		Name: d.Name,
	}).Find(&nodePools).Delete(&nodePools).Error
	if err != nil {
		log.Errorf("Error during deleting saved nodepools: %s", err.Error())
	}

	return nil
}

// BeforeDelete deletes all nodepools to belongs to profile
func (d *AKSProfile) BeforeDelete() error {
	log.Info("BeforeDelete aks profile... delete all nodepool")

	var nodePools []*AKSNodePoolProfile
	return config.DB().Where(AKSNodePoolProfile{
		Name: d.Name,
	}).Find(&nodePools).Delete(&nodePools).Error
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
			nodePools[np.NodeName] = &aks.NodePoolCreate{
				Autoscaling:      np.Autoscaling,
				MinCount:         np.MinCount,
				MaxCount:         np.MaxCount,
				Count:            np.Count,
				NodeInstanceType: np.NodeInstanceType,
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
				nodePools = append(nodePools, &AKSNodePoolProfile{
					Autoscaling:      np.Autoscaling,
					MinCount:         np.MinCount,
					MaxCount:         np.MaxCount,
					Count:            np.Count,
					NodeInstanceType: np.NodeInstanceType,
					Name:             d.Name,
					NodeName:         name,
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
