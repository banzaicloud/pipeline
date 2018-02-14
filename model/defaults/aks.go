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
	database := model.GetDB()
	database.First(&d)
	return database.RowsAffected != int64(0)
}

func (d *AKSProfile) GetType() string {
	return constants.Azure
}

func (d *AKSProfile) GetDefaultProfile() *components.ClusterProfileRespone {
	loadFirst(&d)

	return &components.ClusterProfileRespone{
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
