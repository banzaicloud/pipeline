package model

import (
	"time"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/database"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/amazon"
	"github.com/banzaicloud/pipeline/pkg/cluster/azure"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/pkg/cluster/google"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Simple init for logging
func init() {
	log = config.Logger()
}

// SQL table names
const (
	ProfileTableName              = "oracle_profiles"
	ProfileNodePoolTableName      = "oracle_profiles_nodepools"
	ProfileNodePoolLabelTableName = "oracle_profiles_nodepools_labels"
)

// Profile describes the Oracle cluster profile model
type Profile struct {
	ID        uint   `gorm:"primary_key"`
	Name      string `gorm:"unique_index:idx_modelid_name"`
	Location  string `gorm:"default:'eu-frankfurt-1'"`
	Version   string `gorm:"default:'v1.10.3'"`
	NodePools []*ProfileNodePool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProfileNodePool describes Oracle node pool profile model of a cluster
type ProfileNodePool struct {
	ID        uint   `gorm:"primary_key"`
	Name      string `gorm:"unique_index:idx_modelid_name"`
	Count     uint   `gorm:"default:'1'"`
	Image     string `gorm:"default:'Oracle-Linux-7.4'"`
	Shape     string `gorm:"default:'VM.Standard1.1'"`
	Version   string `gorm:"default:'v1.10.3'"`
	Labels    []*ProfileNodePoolLabel
	ProfileID uint `gorm:"unique_index:idx_modelid_name; foreignKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProfileNodePoolLabel stores labels for node pools
type ProfileNodePoolLabel struct {
	ID                uint   `gorm:"primary_key"`
	Name              string `gorm:"unique_index:idx_name"`
	Value             string
	ProfileNodePoolID uint `gorm:"unique_index:idx_name"`
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
	database.GetDB().Preload("NodePools.Labels").Find(&Profiles)

	return Profiles
}

// GetProfileByName load a Profile from database by it's name and eager load node pools
func GetProfileByName(name string) (Profile, error) {

	var profile Profile
	err := database.GetDB().Where(Profile{Name: name}).Preload("NodePools.Labels").First(&profile).Error

	return profile, err
}

// SaveInstance saves cluster profile into database
func (d *Profile) SaveInstance() error {
	return database.GetDB().Save(d).Error
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *Profile) IsDefinedBefore() bool {
	return database.GetDB().First(&d, Profile{Name: d.Name}).RowsAffected != int64(0)
}

// GetType returns profile's cloud type
func (d *Profile) GetType() string {
	return pkgCluster.Oracle
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
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Eks    *eks.ClusterProfileEks       `json:"eks,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
			Oracle *oracle.Cluster              `json:"oracle,omitempty"`
		}{
			Oracle: &oracle.Cluster{
				Version:   d.Version,
				NodePools: nodePools,
			},
		},
	}
}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *Profile) UpdateProfile(r *pkgCluster.ClusterProfileRequest, withSave bool) error {

	if r != nil {

		s := r.Properties.Oracle

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
	return database.GetDB().Delete(&d).Error
}

// BeforeDelete deletes all nodepools to belongs to profile
func (d *Profile) BeforeDelete() error {
	log.Info("BeforeDelete oke profile... delete all nodepool")

	var nodePools []*ProfileNodePool
	return database.GetDB().Where(ProfileNodePool{
		ProfileID: d.ID,
	}).Find(&nodePools).Delete(&nodePools).Error
}

// BeforeDelete deletes all labels belongs to the nodepool
func (d *ProfileNodePool) BeforeDelete() error {
	log.Info("BeforeDelete oke nodepool... delete all labels")

	var nodePoolLabels []*ProfileNodePoolLabel

	return database.GetDB().Where(ProfileNodePoolLabel{
		ProfileNodePoolID: d.ID,
	}).Find(&nodePoolLabels).Delete(&nodePoolLabels).Error
}

// BeforeSave clears nodepools
func (d *Profile) BeforeSave() error {
	log.Info("BeforeSave oke profile...")

	if d.ID == 0 {
		return nil
	}

	var nodePools []*ProfileNodePool
	err := database.GetDB().Where(ProfileNodePool{
		ProfileID: d.ID,
	}).Find(&nodePools).Delete(&nodePools).Error
	if err != nil {
		log.Errorf("Error during deleting saved nodepools: %s", err.Error())
	}

	return nil
}
