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
	"github.com/banzaicloud/pipeline/pkg/cluster/gke"
)

// GKEProfile describes a Google cluster profile
type GKEProfile struct {
	DefaultModel
	Location      string                `gorm:"default:'us-central1-a'"`
	NodeVersion   string                `gorm:"default:'1.10'"`
	MasterVersion string                `gorm:"default:'1.10'"`
	NodePools     []*GKENodePoolProfile `gorm:"foreignkey:Name"`
}

// GKENodePoolProfile describes a Google cluster profile's nodepools
type GKENodePoolProfile struct {
	ID               uint   `gorm:"primary_key"`
	Autoscaling      bool   `gorm:"default:false"`
	MinCount         int    `gorm:"default:1"`
	MaxCount         int    `gorm:"default:2"`
	Count            int    `gorm:"default:1"`
	NodeInstanceType string `gorm:"default:'n1-standard-1'"`
	Name             string `gorm:"unique_index:idx_name_node_name"`
	NodeName         string `gorm:"unique_index:idx_name_node_name"`
	Preemptible      bool   `gorm:"default:false"`
}

// TableName overrides GKEProfile's table name
func (GKEProfile) TableName() string {
	return DefaultGKEProfileTableName
}

// TableName overrides GKENodePoolProfile's table name
func (GKENodePoolProfile) TableName() string {
	return DefaultGKENodePoolProfileTableName
}

// AfterFind loads nodepools to profile
func (d *GKEProfile) AfterFind() error {
	log.Info("AfterFind gke profile... load node pools")
	return config.DB().Where(GKENodePoolProfile{Name: d.Name}).Find(&d.NodePools).Error
}

// BeforeSave clears nodepools
func (d *GKEProfile) BeforeSave() error {
	log.Info("BeforeSave gke profile...")

	var nodePools []*GKENodePoolProfile
	err := config.DB().Where(GKENodePoolProfile{
		Name: d.Name,
	}).Find(&nodePools).Delete(&nodePools).Error
	if err != nil {
		log.Errorf("Error during deleting saved nodepools: %s", err.Error())
	}

	return nil
}

// BeforeDelete deletes all nodepools to belongs to profile
func (d *GKEProfile) BeforeDelete() error {
	log.Info("BeforeDelete gke profile... delete all nodepool")

	var nodePools []*GKENodePoolProfile
	return config.DB().Where(GKENodePoolProfile{
		Name: d.Name,
	}).Find(&nodePools).Delete(&nodePools).Error
}

// SaveInstance saves cluster profile into database
func (d *GKEProfile) SaveInstance() error {
	return save(d)
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *GKEProfile) IsDefinedBefore() bool {
	return config.DB().First(&d).RowsAffected != int64(0)
}

// GetCloud returns profile's cloud type
func (d *GKEProfile) GetCloud() string {
	return pkgCluster.Google
}

// GetDistribution returns profile's distribution type
func (d *GKEProfile) GetDistribution() pkgCluster.DistributionID {
	return pkgCluster.GKE
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *GKEProfile) GetProfile() *pkgCluster.ClusterProfileResponse {

	nodePools := make(map[string]*gke.NodePool)
	if d.NodePools != nil {
		for _, np := range d.NodePools {
			nodePools[np.NodeName] = &gke.NodePool{
				Autoscaling:      np.Autoscaling,
				MinCount:         np.MinCount,
				MaxCount:         np.MaxCount,
				Count:            np.Count,
				NodeInstanceType: np.NodeInstanceType,
				Preemptible:      np.Preemptible,
			}
		}
	}

	return &pkgCluster.ClusterProfileResponse{
		Name:     d.DefaultModel.Name,
		Location: d.Location,
		Cloud:    pkgCluster.Google,
		Properties: &pkgCluster.ClusterProfileProperties{
			GKE: &gke.ClusterProfileGKE{
				Master: &gke.Master{
					Version: d.MasterVersion,
				},
				NodeVersion: d.NodeVersion,
				NodePools:   nodePools,
			},
		},
	}
}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *GKEProfile) UpdateProfile(r *pkgCluster.ClusterProfileRequest, withSave bool) error {

	if len(r.Location) != 0 {
		d.Location = r.Location
	}

	if r.Properties.GKE != nil {

		if len(r.Properties.GKE.NodeVersion) != 0 {
			d.NodeVersion = r.Properties.GKE.NodeVersion
		}

		if len(r.Properties.GKE.NodePools) != 0 {

			var nodePools []*GKENodePoolProfile
			for name, np := range r.Properties.GKE.NodePools {
				nodePools = append(nodePools, &GKENodePoolProfile{
					Autoscaling:      np.Autoscaling,
					MinCount:         np.MinCount,
					MaxCount:         np.MaxCount,
					Count:            np.Count,
					NodeInstanceType: np.NodeInstanceType,
					Name:             d.Name,
					NodeName:         name,
					Preemptible:      np.Preemptible,
				})
			}

			d.NodePools = nodePools
		}

		if r.Properties.GKE.Master != nil {
			d.MasterVersion = r.Properties.GKE.Master.Version
		}
	}

	if withSave {
		return d.SaveInstance()
	}
	d.Name = r.Name
	return nil
}

// DeleteProfile deletes cluster profile from database
func (d *GKEProfile) DeleteProfile() error {
	return config.DB().Delete(&d).Error
}
