package defaults

import (
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/google"
)

type AKSProfile struct {
	DefaultModel
	Location          string `gorm:"default:'eastus'"`
	NodeInstanceType  string `gorm:"default:'Standard_D2_v2'"`
	AgentCount        int    `gorm:"default:1"`
	AgentName         string `gorm:"default:'agentpool1'"`
	KubernetesVersion string `gorm:"default:'1.8.2'"`
}

func (*AKSProfile) TableName() string {
	return defaultAzureProfileTablaName
}

func (d *AKSProfile) SaveInstance() error {
	return save(d)
}

func (d *AKSProfile) IsDefinedBefore() bool {
	return model.GetDB().First(&d).RowsAffected != int64(0)
}

func (d *AKSProfile) GetType() string {
	return constants.Azure
}

func (d *AKSProfile) GetProfile() *components.ClusterProfileRespone {
	loadFirst(&d)

	return &components.ClusterProfileRespone{
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

func (d *AKSProfile) UpdateProfile(r *components.ClusterProfileRequest) error {
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
	return d.SaveInstance()
}

func (d *AKSProfile) DeleteProfile() error {
	return model.GetDB().Delete(&d).Error
}
