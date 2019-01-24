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

package cluster

import (
	"fmt"
	"regexp"
	"strings"

	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// Cluster describes Pipeline's Oracle fields of a Create/Update request
type Cluster struct {
	Version   string               `json:"version" yaml:"version"`
	NodePools map[string]*NodePool `json:"nodePools,omitempty" yaml:"nodePools,omitempty"`

	vcnID       string
	lbSubnetID1 string
	lbSubnetID2 string
}

// NodePool describes Oracle's node fields of a Create/Update request
type NodePool struct {
	Version string            `json:"version,omitempty" yaml:"version,omitempty"`
	Count   uint              `json:"count,omitempty" yaml:"count,omitempty"`
	Labels  map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Image   string            `json:"image,omitempty" yaml:"image,omitempty"`
	Shape   string            `json:"shape,omitempty" yaml:"shape,omitempty"`

	subnetIds         []string
	quantityPerSubnet uint
}

// SetVCNID sets VCNID
func (c *Cluster) SetVCNID(id string) {

	c.vcnID = id
}

// GetVCNID gets VCNID
func (c *Cluster) GetVCNID() (id string) {

	return c.vcnID
}

// SetLBSubnetID1 sets LBSubnetID1
func (c *Cluster) SetLBSubnetID1(id string) {

	c.lbSubnetID1 = id
}

// GetLBSubnetID1 gets LBSubnetID1
func (c *Cluster) GetLBSubnetID1() (id string) {

	return c.lbSubnetID1
}

// SetLBSubnetID2 sets LBSubnetID2
func (c *Cluster) SetLBSubnetID2(id string) {

	c.lbSubnetID2 = id
}

// GetLBSubnetID2 gets LBSubnetID2
func (c *Cluster) GetLBSubnetID2() (id string) {

	return c.lbSubnetID2
}

// SetQuantityPerSubnet sets QuantityPerSubnet
func (np *NodePool) SetQuantityPerSubnet(q uint) {

	np.quantityPerSubnet = q
}

// GetQuantityPerSubnet gets QuantityPerSubnet
func (np *NodePool) GetQuantityPerSubnet() (q uint) {

	return np.quantityPerSubnet
}

// SetSubnetIDs sets SubnetIDs
func (np *NodePool) SetSubnetIDs(ids []string) {

	np.subnetIds = ids
}

// GetSubnetIDs gets SubnetIDs
func (np *NodePool) GetSubnetIDs() (ids []string) {

	return np.subnetIds
}

// AddDefaults adds default values to the request
func (c *Cluster) AddDefaults() error {

	if c == nil {
		return nil
	}

	// set default version
	if len(c.Version) == 0 {
		c.Version = defaultVersion
	}

	for name, np := range c.NodePools {

		if len(np.Labels) == 0 {
			np.Labels = make(map[string]string)
		}

		np.Labels[pkgCommon.LabelKey] = name

		// set default image
		if len(np.Image) == 0 {
			np.Image = defaultImage
		}

		// set default version
		if len(np.Version) == 0 {
			np.Version = defaultVersion
		}

	}

	return nil
}

// Validate validates Oracle cluster create request
func (c *Cluster) Validate(update bool) error {

	if c == nil {
		return fmt.Errorf("Oracle is <nil>")
	}

	if !isValidVersion(c.Version) {
		return fmt.Errorf("Invalid k8s version: %s", c.Version)
	}

	if len(c.NodePools) < 1 {
		return fmt.Errorf("At least 1 node pool must be specified")
	}

	for name, nodePool := range c.NodePools {
		if nodePool.Version != c.Version {
			return fmt.Errorf("NodePool[%s]: Different k8s versions were specified for master and nodes", name)
		}
		if nodePool.Image == "" && !update {
			return fmt.Errorf("NodePool[%s]: Node image must be specified", name)
		}
		if nodePool.Shape == "" && !update {
			return fmt.Errorf("NodePool[%s]: Node shape must be specified", name)
		}

		for k := range nodePool.Labels {
			if strings.Contains(k, pkgCommon.PipelineSpecificLabelsCommonPart) {
				return pkgErrors.ErrorNodePoolLabelClashesWithPipelineLabel
			}
		}
	}

	return nil
}

// isValidVersion validates the given K8S version
func isValidVersion(version string) bool {

	isOk, _ := regexp.MatchString("^v\\d+\\.\\d+\\.\\d+", version)
	return isOk
}
