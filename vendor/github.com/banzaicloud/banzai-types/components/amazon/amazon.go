package amazon

import (
	"github.com/banzaicloud/banzai-types/constants"
	"errors"
)

type CreateClusterAmazon struct {
	Node   *CreateAmazonNode   `json:"node,omitempty"`
	Master *CreateAmazonMaster `json:"master,omitempty"`
}

type CreateAmazonMaster struct {
	InstanceType string `json:"instanceType"`
	Image        string `json:"image"`
}

type CreateAmazonNode struct {
	SpotPrice string `json:"spotPrice"`
	MinCount  int    `json:"minCount"`
	MaxCount  int    `json:"maxCount"`
	Image     string `json:"image"`
}

type UpdateClusterAmazon struct {
	*UpdateAmazonNode `json:"node,omitempty"`
}

type UpdateAmazonNode struct {
	MinCount int `json:"minCount"`
	MaxCount int `json:"maxCount"`
}

// Validate validates amazon cluster create request
func (amazon *CreateClusterAmazon) Validate() error {
	if amazon == nil {
		return errors.New("Required field 'amazon' is empty.")
	}
	if amazon.Master == nil {
		msg := "Required field 'master' is empty."
		return errors.New(msg)
	}
	if amazon.Master.Image == "" {
		amazon.Master.Image = constants.AmazonDefaultMasterImage
	}

	if amazon.Master.InstanceType == "" {
		amazon.Master.InstanceType = constants.AmazonDefaultMasterInstanceType
	}

	// ---- [ Node check ] ---- //
	if amazon.Node == nil {
		msg := "Required field 'node' is empty."
		return errors.New(msg)
	}

	// ---- [ Node image check ] ---- //
	if len(amazon.Node.Image) == 0 {
		amazon.Node.Image = constants.AmazonDefaultNodeImage
	}

	// ---- [ Node min count check ] ---- //
	if amazon.Node.MinCount == 0 {
		amazon.Node.MinCount = constants.AmazonDefaultNodeMinCount
	}

	// ---- [ Node max count check ] ---- //
	if amazon.Node.MaxCount == 0 {
		amazon.Node.MaxCount = constants.AmazonDefaultNodeMaxCount
	}

	// ---- [ Node min count <= max count check ] ---- //
	if amazon.Node.MaxCount < amazon.Node.MinCount {
		return errors.New("maxCount must be greater than mintCount")
	}

	// ---- [ Node spot price ] ---- //
	if len(amazon.Node.SpotPrice) == 0 {
		amazon.Node.SpotPrice = constants.AmazonDefaultNodeSpotPrice
	}

	return nil
}

// ValidateAmazonRequest validates the update request (only amazon part). If any of the fields is missing, the method fills
// with stored data.
// func (r *UpdateClusterRequest) ValidateAmazonRequest(defaultValue ClusterSimple) (bool, string) {
func (a *UpdateClusterAmazon) Validate() error {

	// ---- [ Amazon field check ] ---- //
	if a == nil {
		return errors.New("'amazon' field is empty")
	}

	// ---- [ Node max count > min count check ] ---- //
	if a.UpdateAmazonNode.MaxCount < a.UpdateAmazonNode.MinCount {
		return errors.New("maxCount must be greater than mintCount")
	}

	return nil
}

type ClusterProfileAmazon struct {
	Master *AmazonProfileMaster `json:"master,omitempty"`
	Node   *AmazonProfileNode   `json:"node,omitempty"`
}

type AmazonProfileMaster struct {
	InstanceType string `json:"instanceType"`
	Image        string `json:"image"`
}

type AmazonProfileNode struct {
	SpotPrice string `json:"spotPrice"`
	MinCount  int    `json:"minCount"`
	MaxCount  int    `json:"maxCount"`
	Image     string `json:"image"`
}
