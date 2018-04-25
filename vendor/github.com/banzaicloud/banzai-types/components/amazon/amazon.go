package amazon

import (
	"github.com/banzaicloud/banzai-types/constants"
)

type CreateClusterAmazon struct {
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
	Master    *CreateAmazonMaster  `json:"master,omitempty"`
}

type CreateAmazonMaster struct {
	InstanceType string `json:"instanceType"`
	Image        string `json:"image"`
}

type NodePool struct {
	InstanceType string `json:"instanceType"`
	SpotPrice    string `json:"spotPrice"`
	MinCount     int    `json:"minCount"`
	MaxCount     int    `json:"maxCount"`
	Image        string `json:"image"`
}

type UpdateClusterAmazon struct {
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}

func (a *NodePool) Validate() error {
	// ---- [ Node image check ] ---- //
	if len(a.InstanceType) == 0 {
		return constants.ErrorAmazonInstancetypeFieldIsEmpty
	}

	// ---- [ Node image check ] ---- //
	if len(a.Image) == 0 {
		return constants.ErrorAmazonImageFieldIsEmpty
	}

	// ---- [ Node min count check ] ---- //
	if a.MinCount == 0 {
		a.MinCount = constants.AmazonDefaultNodeMinCount
	}

	// ---- [ Node max count check ] ---- //
	if a.MaxCount == 0 {
		a.MaxCount = constants.AmazonDefaultNodeMaxCount
	}

	// ---- [ Node min count <= max count check ] ---- //
	if a.MaxCount < a.MinCount {
		return constants.ErrorAmazonMinMaxFieldError
	}

	// ---- [ Node spot price ] ---- //
	if len(a.SpotPrice) == 0 {
		a.SpotPrice = constants.AmazonDefaultNodeSpotPrice
	}

	return nil
}

// Validate validates amazon cluster create request
func (amazon *CreateClusterAmazon) Validate() error {
	if amazon == nil {
		return constants.ErrorAmazonFieldIsEmpty
	}
	if amazon.Master == nil {
		return constants.ErrorAmazonMasterFieldIsEmpty
	}
	if amazon.Master.Image == "" {
		return constants.ErrorAmazonImageFieldIsEmpty
	}

	if amazon.Master.InstanceType == "" {
		amazon.Master.InstanceType = constants.AmazonDefaultMasterInstanceType
	}

	if len(amazon.NodePools) == 0 {
		return constants.ErrorAmazonNodePoolFieldIsEmpty
	}

	for _, np := range amazon.NodePools {
		if err := np.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ValidateAmazonRequest validates the update request (only amazon part). If any of the fields is missing, the method fills
// with stored data.
// func (r *UpdateClusterRequest) ValidateAmazonRequest(defaultValue ClusterSimple) (bool, string) {
func (a *UpdateClusterAmazon) Validate() error {

	// ---- [ Amazon field check ] ---- //
	if a == nil {
		return constants.ErrorAmazonFieldIsEmpty
	}

	if len(a.NodePools) == 0 {
		return constants.ErrorAmazonNodePoolFieldIsEmpty
	}

	for _, np := range a.NodePools {
		if err := np.Validate(); err != nil {
			return err
		}
	}

	return nil
}

type ClusterProfileAmazon struct {
	Master    *AmazonProfileMaster `json:"master,omitempty"`
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}

type AmazonProfileMaster struct {
	InstanceType string `json:"instanceType"`
	Image        string `json:"image"`
}
