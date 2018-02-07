package google

import (
	"github.com/banzaicloud/banzai-types/utils"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/pkg/errors"
)

type CreateClusterGoogle struct {
	Project string        `json:"project"`
	Node    *GoogleNode   `json:"node,omitempty"`
	Master  *GoogleMaster `json:"master,omitempty"`
}

type GoogleMaster struct {
	Version string `json:"version"`
}

type GoogleNode struct {
	Count   int    `json:"count"`
	Version string `json:"version"`
}

type UpdateClusterGoogle struct {
	*GoogleNode   `json:"node,omitempty"`
	*GoogleMaster `json:"master,omitempty"`
}

func (g *CreateClusterGoogle) Validate() (bool, *error) {
	utils.LogInfo(constants.TagValidateCreateCluster, "Start validate create request (google)")

	if g == nil {
		utils.LogInfo(constants.TagValidateCreateCluster, "Google is <nil>")
		err := errors.New("")
		return false, &err
	}

	if len(g.Project) == 0 {
		msg := "Project id is empty"
		utils.LogInfo(constants.TagValidateCreateCluster, msg)
		err := errors.New(msg)
		return false, &err
	}

	if g.Node == nil {
		utils.LogInfo(constants.TagValidateCreateCluster, "Node is <null>")
		g.Node = &GoogleNode{
			Count: 1,
		}
	}

	if g.Master == nil {
		utils.LogInfo(constants.TagValidateCreateCluster, "Master is <null>")
		g.Master = &GoogleMaster{}
	}

	if g.Node.Count == 0 {
		utils.LogInfo(constants.TagValidateCreateCluster, "Node count set to default value:", constants.GoogleDefaultNodeCount)
		g.Node.Count = constants.GoogleDefaultNodeCount
	}

	return true, nil
}

// Validate validates the update request (only google part). If any of the fields is missing, the method fills
// with stored data.
func (a *UpdateClusterGoogle) Validate() error {

	utils.LogInfo(constants.TagValidateCreateCluster, "Start validate update request (google)")

	// ---- [ Google field check ] ---- //
	if a == nil {
		utils.LogInfo(constants.TagValidateCreateCluster, "'google' field is empty")
		return errors.New("'google' field is empty")
	}

	return nil
}
