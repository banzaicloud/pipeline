package azure

import (
	"github.com/banzaicloud/banzai-types/utils"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/components/database"
)

type Values struct {
	Value []Value `json:"value"`
}

type Value struct {
	Id         string     `json:"id"`
	Location   string     `json:"location"`
	Name       string     `json:"name"`
	Properties Properties `json:"properties"`
}

type Properties struct {
	ProvisioningState string    `json:"provisioningState"`
	AgentPoolProfiles []Profile `json:"agentPoolProfiles"`
	Fqdn string  `json:"fqdn"`
}

type Profile struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type ResponseWithValue struct {
	StatusCode int   `json:"status_code"`
	Value      Value `json:"message,omitempty"`
}

type ListResponse struct {
	StatusCode int    `json:"status_code"`
	Value      Values `json:"message"`
}

type CreateClusterAzure struct {
	Node *CreateAzureNode `json:"node,omitempty"`
}

type CreateAzureNode struct {
	ResourceGroup     string `json:"resourceGroup"`
	AgentCount        int    `json:"agentCount"`
	AgentName         string `json:"agentName"`
	KubernetesVersion string `json:"kubernetesVersion"`
}

type UpdateClusterAzure struct {
	*UpdateAzureNode `json:"node,omitempty"`
}

type UpdateAzureNode struct {
	AgentCount int `json:"agentCount"`
}

// Validate validates azure cluster create request
func (azure *CreateClusterAzure) Validate() (bool, string) {

	utils.LogInfo(constants.TagValidateCreateCluster, "Start validate create request (azure)")

	if azure == nil {
		utils.LogInfo(constants.TagValidateCreateCluster, "Azure is <nil>")
		return false, ""
	}

	if azure == nil {
		msg := "Required field 'azure' is empty."
		utils.LogInfo(constants.TagValidateCreateCluster, msg)
		return false, msg
	}

	// ---- [ Node check ] ---- //
	if azure.Node == nil {
		msg := "Required field 'node' is empty."
		utils.LogInfo(constants.TagValidateCreateCluster, msg)
		return false, msg
	}

	if len(azure.Node.ResourceGroup) == 0 {
		msg := "Required field 'resourceGroup' is empty."
		utils.LogInfo(constants.TagValidateCreateCluster, msg)
		return false, msg
	}

	if azure.Node.AgentCount == 0 {
		utils.LogInfo(constants.TagValidateCreateCluster, "Node agentCount set to default value: ", constants.AzureDefaultAgentCount)
		azure.Node.AgentCount = constants.AzureDefaultAgentCount
	}

	if len(azure.Node.AgentName) == 0 {
		utils.LogInfo(constants.TagValidateCreateCluster, "Node agentName set to default value: ", constants.AzureDefaultAgentName)
		azure.Node.AgentName = constants.AzureDefaultAgentName
	}

	if len(azure.Node.KubernetesVersion) == 0 {
		utils.LogInfo(constants.TagValidateCreateCluster, "Node kubernetesVersion set to default value: ", constants.AzureDefaultKubernetesVersion)
		azure.Node.KubernetesVersion = constants.AzureDefaultKubernetesVersion
	}

	return true, ""
}

// ValidateAzureRequest validates the update request (only azure part). If any of the fields is missing, the method fills
// with stored data.
// func (r *UpdateClusterRequest) ValidateAzureRequest(defaultValue components.ClusterSimple) (bool, string) {
func (a *UpdateClusterAzure) Validate(defaultValue database.ClusterSimple) (bool, string) {

	utils.LogInfo(constants.TagValidateCreateCluster, "Start validate update request (azure)")

	defAzureNode := &UpdateAzureNode{
		AgentCount: defaultValue.Azure.AgentCount,
	}

	// ---- [ Azure field check ] ---- //
	if a == nil {
		utils.LogInfo(constants.TagValidateCreateCluster, "'azure' field is empty")
		return false, "'azure' field is empty"
	}

	// ---- [ Node check ] ---- //
	if a.UpdateAzureNode == nil {
		utils.LogInfo(constants.TagValidateCreateCluster, "'node' field is empty. Load it from stored data.")
		a.UpdateAzureNode = defAzureNode
	}

	// ---- [ Node - Agent count check] ---- //
	if a.AgentCount == 0 {
		def := defaultValue.Azure.AgentCount
		utils.LogInfo(constants.TagValidateCreateCluster, "Node agentCount set to default value: ", def)
		a.AgentCount = def
	}

	// create update request struct with the stored data to check equality
	preCl := &UpdateClusterAzure{
		UpdateAzureNode: defAzureNode,
	}

	utils.LogInfo(constants.TagValidateUpdateCluster, "Check stored & updated cluster equals")

	// check equality
	return utils.IsDifferent(a, preCl, constants.TagValidateUpdateCluster)
}

func (r *ResponseWithValue) String() string {
	return utils.Convert2Json(r)
}

func (r *ResponseWithValue) Update(code int, Value Value) {
	r.Value = Value
	r.StatusCode = code
}

func (v *Value) String() string {
	return utils.Convert2Json(v)
}

func (v *Values) String() string {
	return utils.Convert2Json(v)
}

func (l *ListResponse) String() string {
	return utils.Convert2Json(l)
}

func (p *Properties) String() string {
	return utils.Convert2Json(p)
}

func (p *Profile) String() string {
	return utils.Convert2Json(p)
}

type Config struct {
	Location string `json:"location"`
	Name     string `json:"name"`
	Properties struct {
		KubeConfig string `json:"kubeConfig"`
	} `json:"properties"`
}

func (c *Config) String() string {
	return utils.Convert2Json(c)
}
