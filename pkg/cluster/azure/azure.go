package azure

import (
	"errors"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// ### [ Constants to Azure cluster default values ] ### //
const (
	DefaultAgentName         = "agentpool1"
	DefaultKubernetesVersion = "1.9.2"
)

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
		return pkgErrors.ErrorAzureFieldIsEmpty
	}

	// ---- [ NodePool check ] ---- //
	if azure.NodePools == nil {
		return pkgErrors.ErrorNodePoolEmpty
	}

	if len(azure.ResourceGroup) == 0 {
		return pkgErrors.ErrorResourceGroupRequired
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
			np.Count = pkgCommon.DefaultNodeMinCount
		}

		if len(np.NodeInstanceType) == 0 {
			return pkgErrors.ErrorInstancetypeFieldIsEmpty
		}
	}

	if len(azure.KubernetesVersion) == 0 {
		azure.KubernetesVersion = DefaultKubernetesVersion
	}

	return nil
}

// Validate validates the update request (only azure part). If any of the fields is missing, the method fills
// with stored data.
func (a *UpdateClusterAzure) Validate() error {
	// ---- [ Azure field check ] ---- //
	if a == nil {
		return errors.New("'azure' field is empty") // todo move to errors
	}

	return nil
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
