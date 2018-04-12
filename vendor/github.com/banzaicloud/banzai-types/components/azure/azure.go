package azure

import (
	"errors"
	"github.com/banzaicloud/banzai-types/constants"
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
	Fqdn              string    `json:"fqdn"`
}

type Profile struct {
	Name   string `json:"name"`
	Count  int    `json:"count"`
	VmSize string `json:"vmSize"`
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
	ResourceGroup     string                `json:"resourceGroup"`
	KubernetesVersion string                `json:"kubernetesVersion"`
	NodePools         *map[string]*NodePool `json:"nodePools,omitempty"`
}

type NodePool struct {
	AgentCount int    `json:"agentCount"`
	VmSize     string `json:"vmSize"`
}

type UpdateClusterAzure struct {
	NodePools *map[string]*NodePool `json:"nodePools,omitempty"`
}

// Validate validates azure cluster create request
func (azure *CreateClusterAzure) Validate() error {

	if azure == nil {
		return errors.New("Azure is <nil>")
	}

	if azure == nil {
		msg := "Required field 'azure' is empty."
		return errors.New(msg)
	}

	// ---- [ Node check ] ---- //
	if azure.NodePools == nil {
		msg := "Required field 'node' is empty."
		return errors.New(msg)
	}

	if len(azure.ResourceGroup) == 0 {
		msg := "Required field 'resourceGroup' is empty."
		return errors.New(msg)
	}

	for name, np := range *azure.NodePools {
		if np.AgentCount == 0 {
			(*azure.NodePools)[name].AgentCount = constants.AzureDefaultAgentCount
		}

		if len(np.VmSize) == 0 {
			return errors.New("required field `vmSize` is empty")
		}
	}

	if len(azure.KubernetesVersion) == 0 {
		azure.KubernetesVersion = constants.AzureDefaultKubernetesVersion
	}

	return nil
}

// ValidateAzureRequest validates the update request (only azure part). If any of the fields is missing, the method fills
// with stored data.
// func (r *UpdateClusterRequest) ValidateAzureRequest(defaultValue components.ClusterSimple) (bool, string) {
func (a *UpdateClusterAzure) Validate() error {
	// ---- [ Azure field check ] ---- //
	if a == nil {
		return errors.New("'azure' field is empty")
	}

	return nil
}

func (r *ResponseWithValue) Update(code int, Value Value) {
	r.Value = Value
	r.StatusCode = code
}

type Config struct {
	Location string `json:"location"`
	Name     string `json:"name"`
	Properties struct {
		KubeConfig string `json:"kubeConfig"`
	} `json:"properties"`
}

type ClusterProfileAzure struct {
	Node *AzureProfileNode `json:"node,omitempty"`
}

type AzureProfileNode struct {
	AgentCount        int    `json:"agentCount"`
	AgentName         string `json:"agentName"`
	KubernetesVersion string `json:"kubernetesVersion"`
}
