package defaults

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/google"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
)

// GKEProfile describes a Google cluster profile
type GKEProfile struct {
	DefaultModel
	Location      string                `gorm:"default:'us-central1-a'"`
	NodeVersion   string                `gorm:"default:'1.9.4-gke.1'"`
	MasterVersion string                `gorm:"default:'1.9.4-gke.1'"`
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
	Name             string `gorm:"unique_index:idx_model_name"`
	NodeName         string `gorm:"unique_index:idx_model_name"`
	ServiceAccount   string
}

// TableName overrides GKEProfile's table name
func (GKEProfile) TableName() string {
	return DefaultGoogleProfileTablaName
}

// TableName overrides GKENodePoolProfile's table name
func (GKENodePoolProfile) TableName() string {
	return DefaultGoogleNodePoolProfileTablaName
}

// AfterFind loads nodepools to profile
func (d *GKEProfile) AfterFind() error {
	log.Info("AfterFind gke profile... load node pools")
	return model.GetDB().Where(GKENodePoolProfile{Name: d.Name}).Find(&d.NodePools).Error
}

// BeforeSave clears nodepools
func (d *GKEProfile) BeforeSave() error {
	log.Info("BeforeSave gke profile...")

	var nodePools []*GKENodePoolProfile
	err := model.GetDB().Where(GKENodePoolProfile{
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
	return model.GetDB().Where(GKENodePoolProfile{
		Name: d.Name,
	}).Find(&nodePools).Delete(&nodePools).Error
}

// SaveInstance saves cluster profile into database
func (d *GKEProfile) SaveInstance() error {
	return save(d)
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *GKEProfile) IsDefinedBefore() bool {
	return model.GetDB().First(&d).RowsAffected != int64(0)
}

// GetType returns profile's cloud type
func (d *GKEProfile) GetType() string {
	return constants.Google
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *GKEProfile) GetProfile() *components.ClusterProfileResponse {

	nodePools := make(map[string]*google.NodePool)
	if d.NodePools != nil {
		for _, np := range d.NodePools {
			nodePools[np.NodeName] = &google.NodePool{
				Autoscaling:      np.Autoscaling,
				MinCount:         np.MinCount,
				MaxCount:         np.MaxCount,
				Count:            np.Count,
				NodeInstanceType: np.NodeInstanceType,
				ServiceAccount:   np.ServiceAccount,
			}
		}
	}

	return &components.ClusterProfileResponse{
		Name:     d.DefaultModel.Name,
		Location: d.Location,
		Cloud:    constants.Google,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		}{
			Google: &google.ClusterProfileGoogle{
				Master: &google.Master{
					Version: d.MasterVersion,
				},
				NodeVersion: d.NodeVersion,
				NodePools:   nodePools,
			},
		},
	}
}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *GKEProfile) UpdateProfile(r *components.ClusterProfileRequest, withSave bool) error {

	if len(r.Location) != 0 {
		d.Location = r.Location
	}

	if r.Properties.Google != nil {

		if len(r.Properties.Google.NodeVersion) != 0 {
			d.NodeVersion = r.Properties.Google.NodeVersion
		}

		if len(r.Properties.Google.NodePools) != 0 {

			var nodePools []*GKENodePoolProfile
			for name, np := range r.Properties.Google.NodePools {
				nodePools = append(nodePools, &GKENodePoolProfile{
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

		if r.Properties.Google.Master != nil {
			d.MasterVersion = r.Properties.Google.Master.Version
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
	return model.GetDB().Delete(&d).Error
}
