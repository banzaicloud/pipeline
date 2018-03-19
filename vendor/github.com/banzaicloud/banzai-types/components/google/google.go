package google

import (
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/pkg/errors"
	"regexp"
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
	Count          int    `json:"count"`
	Version        string `json:"version"`
	ServiceAccount string `json:"serviceAccount"`
}

type UpdateClusterGoogle struct {
	*GoogleNode   `json:"node,omitempty"`
	*GoogleMaster `json:"master,omitempty"`
}

func (g *CreateClusterGoogle) Validate() error {

	if g == nil {
		return errors.New("Google is <nil>")
	}

	if len(g.Project) == 0 {
		msg := "Project id is empty"
		return errors.New(msg)
	}

	if g.Node == nil {
		g.Node = &GoogleNode{
			Count: 1,
		}
	}

	if g.Master == nil {
		g.Master = &GoogleMaster{}
	}

	if !isValidVersion(g.Node.Version) || !isValidVersion(g.Master.Version) {
		return constants.ErrorWrongKubernetesVersion
	}

	if g.Node.Count == 0 {
		g.Node.Count = constants.GoogleDefaultNodeCount
	}

	return nil
}

// Validate validates the update request (only google part). If any of the fields is missing, the method fills
// with stored data.
func (a *UpdateClusterGoogle) Validate() error {

	// ---- [ Google field check ] ---- //
	if a == nil {
		return errors.New("'google' field is empty")
	}

	// check version
	if (a.GoogleMaster != nil && !isValidVersion(a.GoogleMaster.Version)) ||
		(a.GoogleNode != nil && !isValidVersion(a.GoogleNode.Version)) {
		return constants.ErrorWrongKubernetesVersion
	}

	return nil
}

type ClusterProfileGoogle struct {
	Master *GoogleMaster `json:"master,omitempty"`
	Node   *GoogleNode   `json:"node,omitempty"`
}

func isValidVersion(version string) bool {
	if len(version) == 0 {
		return true
	}

	isOk, _ := regexp.MatchString("^[1-9]\\.([8-9]\\d*|[1-9]\\d+)|^[1-9]\\d+\\.|^[2-9]\\.", version)
	return isOk
}
