package google

import (
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/pkg/errors"
	"regexp"
)

// CreateClusterGoogle describes Pipeline's Google fields of a CreateCluster request
type CreateClusterGoogle struct {
	NodeVersion string               `json:"nodeVersion,omitempty"`
	NodePools   map[string]*NodePool `json:"nodePools,omitempty"`
	Master      *Master              `json:"master,omitempty"`
}

// Master describes Google's master fields of a CreateCluster request
type Master struct {
	Version string `json:"version"`
}

// NodePool describes Google's node fields of a CreateCluster/Update request
type NodePool struct {
	Autoscaling      bool   `json:"autoscaling"`
	MinCount         int    `json:"minCount"`
	MaxCount         int    `json:"maxCount"`
	Count            int    `json:"count,omitempty"`
	NodeInstanceType string `json:"instanceType,omitempty"`
	ServiceAccount   string `json:"serviceAccount,omitempty"`
}

// UpdateClusterGoogle describes Google's node fields of an UpdateCluster request
type UpdateClusterGoogle struct {
	NodeVersion string               `json:"nodeVersion,omitempty"`
	NodePools   map[string]*NodePool `json:"nodePools,omitempty"`
	Master      *Master              `json:"master,omitempty"`
}

// CreateGoogleObjectStoreBucketProperties describes Google Object Store Bucket creation request
type CreateGoogleObjectStoreBucketProperties struct {
	Location string `json:"location,required"`
}

// Validate validates Google cluster create request
func (g *CreateClusterGoogle) Validate() error {

	if g == nil {
		return errors.New("Google is <nil>")
	}

	if g.NodePools == nil {
		g.NodePools = map[string]*NodePool{
			pkgCluster.GoogleDefaultNodePoolName: {
				Count: pkgCluster.DefaultNodeMinCount,
			},
		}
	}

	if g.Master == nil {
		g.Master = &Master{}
	}

	if !isValidVersion(g.Master.Version) || !isValidVersion(g.NodeVersion) {
		return pkgErrors.ErrorWrongKubernetesVersion
	}

	if g.Master.Version != g.NodeVersion {
		return pkgErrors.ErrorDifferentKubernetesVersion
	}

	for _, nodePool := range g.NodePools {

		// ---- [ Min & Max count fields are required in case of autoscaling ] ---- //
		if nodePool.Autoscaling {
			if nodePool.MinCount == 0 {
				return pkgErrors.ErrorMinFieldRequiredError
			}
			if nodePool.MaxCount == 0 {
				return pkgErrors.ErrorMaxFieldRequiredError
			}
			if nodePool.MaxCount < nodePool.MinCount {
				return pkgErrors.ErrorNodePoolMinMaxFieldError
			}
		}

		if nodePool.Count == 0 {
			nodePool.Count = pkgCluster.DefaultNodeMinCount
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
		return pkgErrors.ErrorWrongKubernetesVersion
	}

	// check version equality
	if a.Master != nil && a.Master.Version != a.NodeVersion {
		return pkgErrors.ErrorDifferentKubernetesVersion
	}

	// if nodepools are provided in the update request check that it's not empty
	if a.NodePools != nil && len(a.NodePools) == 0 {
		return pkgErrors.ErrorNodePoolNotProvided
	}

	return nil
}

// ClusterProfileGoogle describes an Amazon profile
type ClusterProfileGoogle struct {
	Master      *Master              `json:"master,omitempty"`
	NodeVersion string               `json:"nodeVersion,omitempty"`
	NodePools   map[string]*NodePool `json:"nodePools,omitempty"`
}

// isValidVersion validates the given K8S version
func isValidVersion(version string) bool {
	if len(version) == 0 {
		return true
	}

	isOk, _ := regexp.MatchString("^[1-9]\\.([8-9]\\d*|[1-9]\\d+)|^[1-9]\\d+\\.|^[2-9]\\.", version)
	return isOk
}
