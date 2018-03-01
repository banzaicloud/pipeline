package defaults

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/google"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
)

// AKSProfile describes an Azure cluster profile
type AKSProfile struct {
	DefaultModel
	Location          string `gorm:"default:'eastus'"`
	NodeInstanceType  string `gorm:"default:'Standard_D2_v2'"`
	AgentCount        int    `gorm:"default:1"`
	AgentName         string `gorm:"default:'agentpool1'"`
	KubernetesVersion string `gorm:"default:'1.8.2'"`
}

// TableName overrides AKSProfile's table name
func (AKSProfile) TableName() string {
	return DefaultAzureProfileTablaName
}

// SaveInstance saves cluster profile into database
func (d *AKSProfile) SaveInstance() error {
	return save(d)
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *AKSProfile) IsDefinedBefore() bool {
	return model.GetDB().First(&d).RowsAffected != int64(0)
}

// GetType returns profile's cloud type
func (d *AKSProfile) GetType() string {
	return constants.Azure
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *AKSProfile) GetProfile() *components.ClusterProfileResponse {
	loadFirst(&d)

	return &components.ClusterProfileResponse{
		ProfileName:      d.DefaultModel.Name,
		Location:         d.Location,
		Cloud:            constants.Azure,
		NodeInstanceType: d.NodeInstanceType,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		}{
			Azure: &azure.ClusterProfileAzure{
				Node: &azure.AzureProfileNode{
					AgentCount:        d.AgentCount,
					AgentName:         d.AgentName,
					KubernetesVersion: d.KubernetesVersion,
				},
			},
		},
	}
}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *AKSProfile) UpdateProfile(r *components.ClusterProfileRequest, withSave bool) error {
	if len(r.Location) != 0 {
		d.Location = r.Location
	}

	if len(r.NodeInstanceType) != 0 {
		d.NodeInstanceType = r.NodeInstanceType
	}

	if r.Properties.Azure != nil {
		if r.Properties.Azure.Node != nil {
			if r.Properties.Azure.Node.AgentCount != 0 {
				d.AgentCount = r.Properties.Azure.Node.AgentCount
			}
			if len(r.Properties.Azure.Node.AgentName) != 0 {
				d.AgentName = r.Properties.Azure.Node.AgentName
			}
			if len(r.Properties.Azure.Node.KubernetesVersion) != 0 {
				d.KubernetesVersion = r.Properties.Azure.Node.KubernetesVersion
			}
		}
	}

	if withSave {
		return d.SaveInstance()
	}
	d.Name = r.ProfileName
	return nil
}

// DeleteProfile deletes cluster profile from database
func (d *AKSProfile) DeleteProfile() error {
	return model.GetDB().Delete(&d).Error
}
