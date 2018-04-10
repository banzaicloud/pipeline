package google

import (
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/pkg/errors"
	"regexp"
)

type CreateClusterGoogle struct {
	Project     string                     `json:"project"`
	NodeVersion string                     `json:"nodeVersion,omitempty"`
	NodePools   map[string]*GoogleNodePool `json:"nodePools,omitempty"`
	Master      *GoogleMaster              `json:"master,omitempty"`
}

type GoogleMaster struct {
	Version string `json:"version"`
}

type GoogleNodePool struct {
	Count            int    `json:"count,omitempty"`
	NodeInstanceType string `json:"nodeInstanceType,omitempty"`
	ServiceAccount   string `json:"serviceAccount,omitempty"`
}

type UpdateClusterGoogle struct {
	NodeVersion string                     `json:"nodeVersion,omitempty"`
	NodePools   map[string]*GoogleNodePool `json:"nodePools,omitempty"`
	Master      *GoogleMaster              `json:"master,omitempty"`
}

func (g *CreateClusterGoogle) Validate() error {

	if g == nil {
		return errors.New("Google is <nil>")
	}

	if len(g.Project) == 0 {
		msg := "Project id is empty"
		return errors.New(msg)
	}

	if g.NodePools == nil {
		g.NodePools = map[string]*GoogleNodePool{
			constants.GoogleDefaultNodePoolName: &GoogleNodePool{
				Count: constants.GoogleDefaultNodeCount,
			},
		}
	}

	if g.Master == nil {
		g.Master = &GoogleMaster{}
	}

	if !isValidVersion(g.Master.Version) || !isValidVersion(g.NodeVersion) {
		return constants.ErrorWrongKubernetesVersion
	}

	if g.Master.Version != g.NodeVersion {
		return constants.ErrorDifferentKubernetesVersion
	}

	for _, nodePool := range g.NodePools {
		if nodePool.Count == 0 {
			nodePool.Count = constants.GoogleDefaultNodeCount
		}
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
	if (a.Master != nil && !isValidVersion(a.Master.Version)) || !isValidVersion(a.NodeVersion) {
		return constants.ErrorWrongKubernetesVersion
	}

	// check version equality
	if a.Master != nil && a.Master.Version != a.NodeVersion {
		return constants.ErrorDifferentKubernetesVersion
	}

	// if nodepools are provided in the update request check that it's not empty
	if a.NodePools != nil && len(a.NodePools) == 0 {
		return constants.ErrorNodePoolNotProvided
	}

	return nil
}

type ClusterProfileGoogle struct {
	Master      *GoogleMaster              `json:"master,omitempty"`
	NodeVersion string                     `json:"nodeVersion,omitempty"`
	NodePools   map[string]*GoogleNodePool `json:"nodePools,omitempty"`
}

func isValidVersion(version string) bool {
	if len(version) == 0 {
		return true
	}

	isOk, _ := regexp.MatchString("^[1-9]\\.([8-9]\\d*|[1-9]\\d+)|^[1-9]\\d+\\.|^[2-9]\\.", version)
	return isOk
}
