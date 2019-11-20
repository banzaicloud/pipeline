// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gke

import (
	"regexp"

	"github.com/pkg/errors"

	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// ### [ Constants to Google cluster default values ] ### //
const (
	DefaultNodePoolName = "default-pool"
)

// CreateClusterGKE describes Pipeline's Google fields of a CreateCluster request
type CreateClusterGKE struct {
	NodeVersion string               `json:"nodeVersion,omitempty" yaml:"nodeVersion,omitempty"`
	NodePools   map[string]*NodePool `json:"nodePools,omitempty" yaml:"nodePools,omitempty"`
	Master      *Master              `json:"master,omitempty" yaml:"master,omitempty"`
	Vpc         string               `json:"vpc,omitempty" yaml:"vpc,omitempty"`
	Subnet      string               `json:"subnet,omitempty" yaml:"subnet,omitempty"`
	ProjectId   string               `json:"projectId" yaml:"projectId"`
}

// Master describes Google's master fields of a CreateCluster request
type Master struct {
	Version string `json:"version"`
}

// NodePool describes Google's node fields of a CreateCluster/Update request
type NodePool struct {
	Autoscaling      bool              `json:"autoscaling" yaml:"autoscaling"`
	MinCount         int               `json:"minCount" yaml:"minCount"`
	MaxCount         int               `json:"maxCount" yaml:"maxCount"`
	Count            int               `json:"count,omitempty" yaml:"count,omitempty"`
	NodeInstanceType string            `json:"instanceType,omitempty" yaml:"instanceType,omitempty"`
	Preemptible      bool              `json:"preemptible,omitempty" yaml:"preemptible,omitempty"`
	Labels           map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// UpdateClusterGoogle describes Google's node fields of an UpdateCluster request
type UpdateClusterGoogle struct {
	NodeVersion string               `json:"nodeVersion,omitempty"`
	NodePools   map[string]*NodePool `json:"nodePools,omitempty"`
	Master      *Master              `json:"master,omitempty"`
}

// Validate validates Google cluster create request
func (g *CreateClusterGKE) Validate() error {

	if g == nil {
		return errors.New("Google is <nil>")
	}

	if g.NodePools == nil {
		g.NodePools = map[string]*NodePool{
			DefaultNodePoolName: {
				Count: pkgCommon.DefaultNodeMinCount,
			},
		}
	}

	if g.Master == nil {
		g.Master = &Master{
			Version: g.NodeVersion,
		}
	} else if len(g.NodeVersion) == 0 {
		g.NodeVersion = g.Master.Version
	}

	if !isValidVersion(g.Master.Version) || !isValidVersion(g.NodeVersion) {
		return pkgErrors.ErrorWrongKubernetesVersion
	}

	if g.Master.Version != g.NodeVersion {
		return pkgErrors.ErrorDifferentKubernetesVersion
	}

	if len(g.Vpc) > 0 && g.Vpc != "default" && len(g.Subnet) == 0 {
		return pkgErrors.ErrorGkeSubnetRequiredFieldIsEmpty
	}

	if len(g.Subnet) > 0 && len(g.Vpc) == 0 {
		return pkgErrors.ErrorGkeVPCRequiredFieldIsEmpty
	}

	for _, nodePool := range g.NodePools {

		// ---- [ Min & Max count fields are required in case of auto scaling ] ---- //
		if nodePool.Autoscaling {
			if nodePool.MaxCount == 0 {
				return pkgErrors.ErrorMaxFieldRequiredError
			}
			if nodePool.MaxCount < nodePool.MinCount {
				return pkgErrors.ErrorNodePoolMinMaxFieldError
			}
		}

		if nodePool.Count == 0 {
			nodePool.Count = pkgCommon.DefaultNodeMinCount
		}

		if err := pkgCommon.ValidateNodePoolLabels(nodePool.Labels); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the update request (only gke part). If any of the fields is missing, the method fills
// with stored data.
func (a *UpdateClusterGoogle) Validate() error {

	// ---- [ Google field check ] ---- //
	if a == nil {
		return errors.New("'gke' field is empty")
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

// isValidVersion validates the given K8S version
func isValidVersion(version string) bool {
	if len(version) == 0 {
		return true
	}

	isOk, _ := regexp.MatchString("^[1-9]\\.([8-9]\\d*|[1-9]\\d+)|^[1-9]\\d+\\.|^[2-9]\\.", version)
	return isOk
}
