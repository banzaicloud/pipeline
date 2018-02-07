package amazon

import (
	"github.com/banzaicloud/banzai-types/utils"
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
func (amazon *CreateClusterAmazon) Validate() (bool, string) {

	utils.LogInfo(constants.TagValidateCreateCluster, "Validate create request (amazon)")

	if amazon == nil {
		utils.LogInfo(constants.TagValidateCreateCluster, "Amazon is <nil>")
		return false, ""
	}

	if amazon == nil {
		msg := "Required field 'amazon' is empty."
		utils.LogInfo(constants.TagValidateCreateCluster, msg)
		return false, msg
	}

	// ---- [ Master check ] ---- //
	if amazon.Master == nil {
		msg := "Required field 'master' is empty."
		utils.LogInfo(constants.TagValidateCreateCluster, msg)
		return false, msg
	}

	if len(amazon.Master.Image) == 0 {
		utils.LogInfo(constants.TagValidateCreateCluster, "Master image set to default value:", constants.AmazonDefaultMasterImage)
		amazon.Master.Image = constants.AmazonDefaultMasterImage
	}

	if len(amazon.Master.InstanceType) == 0 {
		utils.LogInfo(constants.TagValidateCreateCluster, "Master instance type set to default value:", constants.AmazonDefaultMasterInstanceType)
		amazon.Master.InstanceType = constants.AmazonDefaultMasterInstanceType
	}

	// ---- [ Node check ] ---- //
	if amazon.Node == nil {
		msg := "Required field 'node' is empty."
		utils.LogInfo(constants.TagValidateCreateCluster, msg)
		return false, msg
	}

	// ---- [ Node image check ] ---- //
	if len(amazon.Node.Image) == 0 {
		utils.LogInfo(constants.TagValidateCreateCluster, "Node image set to default value:", constants.AmazonDefaultNodeImage)
		amazon.Node.Image = constants.AmazonDefaultNodeImage
	}

	// ---- [ Node min count check ] ---- //
	if amazon.Node.MinCount == 0 {
		utils.LogInfo(constants.TagValidateCreateCluster, "Node minCount set to default value:", constants.AmazonDefaultNodeMinCount)
		amazon.Node.MinCount = constants.AmazonDefaultNodeMinCount
	}

	// ---- [ Node max count check ] ---- //
	if amazon.Node.MaxCount == 0 {
		utils.LogInfo(constants.TagValidateCreateCluster, "Node maxCount set to default value:", constants.AmazonDefaultNodeMaxCount)
		amazon.Node.MaxCount = constants.AmazonDefaultNodeMaxCount
	}

	// ---- [ Node min count <= max count check ] ---- //
	if amazon.Node.MaxCount < amazon.Node.MinCount {
		utils.LogInfo(constants.TagValidateCreateCluster, "Node maxCount is lower than minCount")
		return false, "maxCount must be greater than mintCount"
	}

	// ---- [ Node spot price ] ---- //
	if len(amazon.Node.SpotPrice) == 0 {
		utils.LogInfo(constants.TagValidateCreateCluster, "Node spot price set to default value:", constants.AmazonDefaultNodeSpotPrice)
		amazon.Node.SpotPrice = constants.AmazonDefaultNodeSpotPrice
	}

	return true, ""
}

// ValidateAmazonRequest validates the update request (only amazon part). If any of the fields is missing, the method fills
// with stored data.
// func (r *UpdateClusterRequest) ValidateAmazonRequest(defaultValue ClusterSimple) (bool, string) {
func (a *UpdateClusterAmazon) Validate() error {

	utils.LogInfo(constants.TagValidateUpdateCluster, "Validate update request (amazon)")

	// ---- [ Amazon field check ] ---- //
	if a == nil {
		utils.LogInfo(constants.TagValidateUpdateCluster, "'amazon' field is empty")
		return errors.New("'amazon' field is empty")
	}

	// ---- [ Node max count > min count check ] ---- //
	if a.UpdateAmazonNode.MaxCount < a.UpdateAmazonNode.MinCount {
		utils.LogInfo(constants.TagValidateUpdateCluster, "Node maxCount is lower than minCount")
		return errors.New("maxCount must be greater than mintCount")
	}

	return nil
}
