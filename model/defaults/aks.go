package defaults

import (
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/google"
)

type DefaultCreateRequestAKS struct {
	DefaultModel
	Location          string `gorm:"default:'eastus'"`
	NodeInstanceType  string `gorm:"default:'Standard_D2_v2'"`
	AgentCount        int    `gorm:"default:1"`
	AgentName         string `gorm:"default:'agentpool1'"`
	KubernetesVersion string `gorm:"default:'1.8.2'"`
}

func (*DefaultCreateRequestAKS) TableName() string {
	return "azure_default_create"
}

func (d *DefaultCreateRequestAKS) SaveDefaultInstance() error {
	return save(d)
}

func (d *DefaultCreateRequestAKS) IsDefinedBefore() bool {
	database := model.GetDB()
	database.First(&d)
	return d.ID != 0
}

func (d *DefaultCreateRequestAKS) GetType() string {
	return constants.Azure
}

func (d *DefaultCreateRequestAKS) GetDefaultCreateClusterRequest() *components.CreateClusterRequest {
	loadFirst(&d)

	return &components.CreateClusterRequest{
		Name:             "", // todo good??
		Location:         d.Location,
		Cloud:            constants.Azure,
		NodeInstanceType: d.NodeInstanceType,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
		}{
			CreateClusterAzure: &azure.CreateClusterAzure{
				Node: &azure.CreateAzureNode{
					ResourceGroup:     "", // todo good?
					AgentCount:        d.AgentCount,
					AgentName:         d.AgentName,
					KubernetesVersion: d.KubernetesVersion,
				},
			},
		},
	}
}
