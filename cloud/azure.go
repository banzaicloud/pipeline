package cloud

import "github.com/jinzhu/gorm"

type CreateClusterAzure struct {
	gorm.Model
	Node CreateAzureNode `json:"node"`
}

type CreateAzureNode struct {
	gorm.Model
	ResourceGroup     string `json:"resourceGroup"`
	AgentCount        int    `json:"agentCount"`
	AgentName         string `json:"agentName"`
	KubernetesVersion string `json:"kubernetesVersion"`
}
