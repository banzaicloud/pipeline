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
	Location         string `gorm:"default:'us-central1-a'"`
	NodeInstanceType string `gorm:"default:'n1-standard-1'"`
	NodeCount        int    `gorm:"default:1"`
	NodeVersion      string `gorm:"default:'1.9.4-gke.1'"`
	MasterVersion    string `gorm:"default:'1.9.4-gke.1'"`
	ServiceAccount   string
}

// TableName overrides GKEProfile's table name
func (GKEProfile) TableName() string {
	return DefaultGoogleProfileTablaName
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
	loadFirst(&d)

	return &components.ClusterProfileResponse{
		Name:             d.DefaultModel.Name,
		Location:         d.Location,
		Cloud:            constants.Google,
		NodeInstanceType: d.NodeInstanceType,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		}{
			Google: &google.ClusterProfileGoogle{
				NodeVersion: d.NodeVersion,
				NodePools:   nil, // TODO : finish me
				Master: &google.GoogleMaster{
					Version: d.MasterVersion,
				},
			},
		},
	}
}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *GKEProfile) UpdateProfile(r *components.ClusterProfileRequest, withSave bool) error {

	if len(r.Location) != 0 {
		d.Location = r.Location
	}

	if len(r.NodeInstanceType) != 0 {
		d.NodeInstanceType = r.NodeInstanceType
	}

	if r.Properties.Google != nil {

		if len(r.Properties.Google.NodeVersion) != 0 {
			d.NodeVersion = r.Properties.Google.NodeVersion
		}

		// TODO : fix me

		if len(r.Properties.Google.NodePools) > 0 {
			for _, v := range r.Properties.Google.NodePools {
				if v.Count != 0 {
					d.NodeCount = v.Count
				}
				if len(v.ServiceAccount) != 0 {
					d.ServiceAccount = v.ServiceAccount
				}
				break
			}
		}
		/*
			if r.Properties.Google.Node != nil {
				if r.Properties.Google.Node.Count != 0 {
					d.NodeCount = r.Properties.Google.Node.Count
				}

				if len(r.Properties.Google.Node.ServiceAccount) != 0 {
					d.ServiceAccount = r.Properties.Google.Node.ServiceAccount
				}

			}*/

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
