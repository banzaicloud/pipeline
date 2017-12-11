package cloud

import (
	"github.com/sirupsen/logrus"
)

const (
	azureDefaultAgentCount        = 1
	azureDefaultAgentName         = "agentpool1"
	azureDefaultKubernetesVersion = "1.7.7"
)

type CreateClusterAzure struct {
	Node *CreateAzureNode `json:"node"`
}

type UpdateClusterAzure struct {
	*UpdateAzureNode `json:"node"`
}

type CreateAzureNode struct {
	ResourceGroup     string `json:"resourceGroup"`
	AgentCount        int    `json:"agentCount"`
	AgentName         string `json:"agentName"`
	KubernetesVersion string `json:"kubernetesVersion"`
}

type UpdateAzureNode struct {
	AgentCount        int    `json:"agentCount"`
}

type AzureSimple struct {
	CreateClusterSimpleId uint `gorm:"primary_key"`
	ResourceGroup         string
	AgentCount            int
	AgentName             string
	KubernetesVersion     string
}

// TableName sets AzureSimple's table name
func (AzureSimple) TableName() string {
	return tableNameAzureProperties
}

// Validate validates azure cluster create request
func (azure *CreateClusterAzure) Validate(log *logrus.Logger) (bool, string) {

	if azure == nil {
		msg := "Required field 'azure' is empty."
		log.Info(msg)
		return false, msg
	}

	// ---- [ Node check ] ---- //
	if azure.Node == nil {
		msg := "Required field 'node' is empty."
		log.Info(msg)
		return false, msg
	}

	if len(azure.Node.ResourceGroup) == 0 {
		msg := "Required field 'resourceGroup' is empty."
		log.Info(msg)
		return false, msg
	}

	if azure.Node.AgentCount == 0 {
		log.Info("Node agentCount set to default value: ", azureDefaultAgentCount)
		azure.Node.AgentCount = azureDefaultAgentCount
	}

	if len(azure.Node.AgentName) == 0 {
		log.Info("Node agentName set to default value: ", azureDefaultAgentName)
		azure.Node.AgentName = azureDefaultAgentName
	}

	if len(azure.Node.KubernetesVersion) == 0 {
		log.Info("Node kubernetesVersion set to default value: ", azureDefaultKubernetesVersion)
		azure.Node.KubernetesVersion = azureDefaultKubernetesVersion
	}

	return true, ""
}
