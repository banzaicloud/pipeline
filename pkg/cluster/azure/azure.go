package azure

import (
	"errors"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// Values describes a list of Azure clusters
type Values struct {
	Value []Value `json:"value"`
}

// Value describes an Azure cluster
type Value struct {
	Id         string     `json:"id"`
	Location   string     `json:"location"`
	Name       string     `json:"name"`
	Properties Properties `json:"properties"`
}

// Properties describes an Azure cluster properties
type Properties struct {
	ProvisioningState string    `json:"provisioningState"`
	AgentPoolProfiles []Profile `json:"agentPoolProfiles"`
	Fqdn              string    `json:"fqdn"`
}

// Profile describes an Azure agent pool
type Profile struct {
	Name        string `json:"name"`
	Autoscaling bool   `json:"autoscaling"`
	MinCount    int    `json:"minCount"`
	MaxCount    int    `json:"maxCount"`
	Count       int    `json:"count"`
	VmSize      string `json:"vmSize"`
}

// ResponseWithValue describes an Azure cluster
type ResponseWithValue struct {
	StatusCode int   `json:"status_code"`
	Value      Value `json:"message,omitempty"`
}

// ListResponse describes an Azure cluster list
type ListResponse struct {
	StatusCode int    `json:"status_code"`
	Value      Values `json:"message"`
}

// CreateClusterAzure describes Azure fields of a CreateCluster request
type CreateClusterAzure struct {
	ResourceGroup     string                     `json:"resourceGroup"`
	KubernetesVersion string                     `json:"kubernetesVersion"`
	NodePools         map[string]*NodePoolCreate `json:"nodePools,omitempty"`
}

// NodePoolCreate describes Azure's node fields of a CreateCluster request
type NodePoolCreate struct {
	Autoscaling      bool   `json:"autoscaling"`
	MinCount         int    `json:"minCount"`
	MaxCount         int    `json:"maxCount"`
	Count            int    `json:"count"`
	NodeInstanceType string `json:"instanceType"`
}

// NodePoolUpdate describes Azure's node count of a UpdateCluster request
type NodePoolUpdate struct {
	Autoscaling bool `json:"autoscaling"`
	MinCount    int  `json:"minCount"`
	MaxCount    int  `json:"maxCount"`
	Count       int  `json:"count"`
}

// UpdateClusterAzure describes Azure's node fields of an UpdateCluster request
type UpdateClusterAzure struct {
	NodePools map[string]*NodePoolUpdate `json:"nodePools,omitempty"`
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

	// ---- [ NodePool check ] ---- //
	if azure.NodePools == nil {
		msg := "Required field 'nodePools' is empty."
		return errors.New(msg)
	}

	if len(azure.ResourceGroup) == 0 {
		msg := "Required field 'resourceGroup' is empty."
		return errors.New(msg)
	}

	for _, np := range azure.NodePools {

		// ---- [ Min & Max count fields are required in case of autoscaling ] ---- //
		if np.Autoscaling {
			if np.MinCount == 0 {
				return pkgErrors.ErrorMinFieldRequiredError
			}
			if np.MaxCount == 0 {
				return pkgErrors.ErrorMaxFieldRequiredError
			}
			if np.MaxCount < np.MinCount {
				return pkgErrors.ErrorNodePoolMinMaxFieldError
			}
		}

		if np.Count == 0 {
			np.Count = pkgCluster.DefaultNodeMinCount
		}

		if len(np.NodeInstanceType) == 0 {
			return errors.New("required field `NodeInstanceType` is empty")
		}
	}

	if len(azure.KubernetesVersion) == 0 {
		azure.KubernetesVersion = pkgCluster.AzureDefaultKubernetesVersion
	}

	return nil
}

// Validate validates the update request (only azure part). If any of the fields is missing, the method fills
// with stored data.
func (a *UpdateClusterAzure) Validate() error {
	// ---- [ Azure field check ] ---- //
	if a == nil {
		return errors.New("'azure' field is empty")
	}

	return nil
}

// Update updates `ResponseWithValue` with the given response code and value
func (r *ResponseWithValue) Update(code int, Value Value) {
	r.Value = Value
	r.StatusCode = code
}

// Config describes an Azure kubeconfig
type Config struct {
	Location   string `json:"location"`
	Name       string `json:"name"`
	Properties struct {
		KubeConfig string `json:"kubeConfig"`
	} `json:"properties"`
}

// ClusterProfileAzure describes an Azure profile
type ClusterProfileAzure struct {
	KubernetesVersion string                     `json:"kubernetesVersion"`
	NodePools         map[string]*NodePoolCreate `json:"nodePools,omitempty"`
}

// CreateAzureObjectStoreBucketProperties describes an Azure ObjectStore Container Creation request
type CreateAzureObjectStoreBucketProperties struct {
	Location       string `json:"location" binding:"required"`
	StorageAccount string `json:"storageAccount"`
	ResourceGroup  string `json:"resourceGroup"`
}

// BlobStoragePropsForAzure describes the Azure specific properties
type BlobStoragePropsForAzure struct {
	ResourceGroup  string `json:"resourceGroup" binding:"required"`
	StorageAccount string `json:"storageAccount" binding:"required"`
}
