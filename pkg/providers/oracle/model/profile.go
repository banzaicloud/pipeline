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

package model

import (
	"time"

	"github.com/banzaicloud/pipeline/config"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
)

// SQL table names
const (
	ProfileTableName              = "oracle_oke_profiles"
	ProfileNodePoolTableName      = "oracle_oke_profile_node_pools"
	ProfileNodePoolLabelTableName = "oracle_oke_profile_node_pool_labels"
)

// Profile describes the Oracle cluster profile model
type Profile struct {
	ID        uint   `gorm:"primary_key"`
	Name      string `gorm:"unique_index:idx_name"`
	Location  string `gorm:"default:'eu-frankfurt-1'"`
	Version   string `gorm:"default:'v1.10.3'"`
	NodePools []*ProfileNodePool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProfileNodePool describes Oracle node pool profile model of a cluster
type ProfileNodePool struct {
	ID        uint   `gorm:"primary_key"`
	Name      string `gorm:"unique_index:idx_name_profile_id"`
	Count     uint   `gorm:"default:'1'"`
	Image     string `gorm:"default:'Oracle-Linux-7.4'"`
	Shape     string `gorm:"default:'VM.Standard1.1'"`
	Version   string `gorm:"default:'v1.10.3'"`
	Labels    []*ProfileNodePoolLabel
	ProfileID uint `gorm:"unique_index:idx_name_profile_id; foreignKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProfileNodePoolLabel stores labels for node pools
type ProfileNodePoolLabel struct {
	ID                uint   `gorm:"primary_key"`
	Name              string `gorm:"unique_index:idx_name_profile_node_pool_id"`
	Value             string
	ProfileNodePoolID uint `gorm:"unique_index:idx_name_profile_node_pool_id"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TableName overrides Profile table name
func (Profile) TableName() string {
	return ProfileTableName
}

// TableName overrides ProfileNodePool table name
func (ProfileNodePool) TableName() string {
	return ProfileNodePoolTableName
}

// TableName overrides ProfileNodePoolLabel table name
func (ProfileNodePoolLabel) TableName() string {
	return ProfileNodePoolLabelTableName
}

// GetProfiles gets Profiles from database and eager loads node pools
func GetProfiles() []Profile {

	var Profiles []Profile
	config.DB().Preload("NodePools.Labels").Find(&Profiles)

	return Profiles
}

// GetProfileByName load a Profile from database by it's name and eager load node pools
func GetProfileByName(name string) (Profile, error) {

	var profile Profile
	err := config.DB().Where(Profile{Name: name}).Preload("NodePools.Labels").First(&profile).Error

	return profile, err
}

// SaveInstance saves cluster profile into database
func (d *Profile) SaveInstance() error {
	return config.DB().Save(d).Error
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *Profile) IsDefinedBefore() bool {
	return config.DB().First(&d, Profile{Name: d.Name}).RowsAffected != int64(0)
}

// GetCloud returns profile's cloud type
func (d *Profile) GetCloud() string {
	return pkgCluster.Oracle
}

// GetDistribution returns profile's distribution type
func (d *Profile) GetDistribution() pkgCluster.DistributionID {
	return pkgCluster.OKE
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *Profile) GetProfile() *pkgCluster.ClusterProfileResponse {

	nodePools := make(map[string]*oracle.NodePool)
	if d.NodePools != nil {
		for _, np := range d.NodePools {
			nodePools[np.Name] = &oracle.NodePool{
				Version: np.Version,
				Image:   np.Image,
				Count:   np.Count,
				Shape:   np.Shape,
			}
			nodePools[np.Name].Labels = make(map[string]string, 0)
			for _, l := range np.Labels {
				nodePools[np.Name].Labels[l.Name] = l.Value
			}
		}
	}

	return &pkgCluster.ClusterProfileResponse{
		Name:     d.Name,
		Location: d.Location,
		Cloud:    pkgCluster.Oracle,
		Properties: &pkgCluster.ClusterProfileProperties{
			OKE: &oracle.Cluster{
				Version:   d.Version,
				NodePools: nodePools,
			},
		},
	}
}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *Profile) UpdateProfile(r *pkgCluster.ClusterProfileRequest, withSave bool) error {

	if r != nil {

		s := r.Properties.OKE

		d.Version = s.Version
		d.Location = r.Location

		if len(s.NodePools) != 0 {
			var nodePools []*ProfileNodePool
			for name, np := range s.NodePools {
				nodePool := &ProfileNodePool{
					Version: np.Version,
					Count:   np.Count,
					Image:   np.Image,
					Shape:   np.Shape,
					Name:    name,
				}
				for name, value := range np.Labels {
					nodePool.Labels = append(nodePool.Labels, &ProfileNodePoolLabel{
						Name:  name,
						Value: value,
					})
				}
				nodePools = append(nodePools, nodePool)
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
func (d *Profile) DeleteProfile() error {
	return config.DB().Delete(&d).Error
}

// BeforeDelete deletes all nodepools to belongs to profile
func (d *Profile) BeforeDelete() error {
	log.Info("BeforeDelete oracle profile... delete all nodepool")

	var nodePools []*ProfileNodePool
	return config.DB().Where(ProfileNodePool{
		ProfileID: d.ID,
	}).Find(&nodePools).Delete(&nodePools).Error
}

// BeforeDelete deletes all labels belongs to the nodepool
func (d *ProfileNodePool) BeforeDelete() error {
	log.Info("BeforeDelete oracle nodepool... delete all labels")

	var nodePoolLabels []*ProfileNodePoolLabel

	return config.DB().Where(ProfileNodePoolLabel{
		ProfileNodePoolID: d.ID,
	}).Find(&nodePoolLabels).Delete(&nodePoolLabels).Error
}

// BeforeSave clears nodepools
func (d *Profile) BeforeSave() error {
	log.Info("BeforeSave oracle profile...")

	if d.ID == 0 {
		return nil
	}

	var nodePools []*ProfileNodePool
	err := config.DB().Where(ProfileNodePool{
		ProfileID: d.ID,
	}).Find(&nodePools).Delete(&nodePools).Error
	if err != nil {
		log.Errorf("Error during deleting saved nodepools: %s", err.Error())
	}

	return nil
}
