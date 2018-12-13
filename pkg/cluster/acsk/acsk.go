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

package acsk

import (
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// NodePool describes Alibaba's node fields of a CreateCluster/Update request
type NodePool struct {
	InstanceType       string `json:"instanceType"`
	MinCount           int    `json:"minCount"`
	MaxCount           int    `json:"maxCount"`
}

type NodePools map[string]*NodePool

// CreateClusterACSK
type CreateClusterACSK struct {
	RegionID                 string    `json:"regionId" yaml:"regionId"`
	ZoneID                   string    `json:"zoneId" yaml:"zoneId"`
	MasterInstanceType       string    `json:"masterInstanceType,omitempty" yaml:"masterInstanceType,omitempty"`
	MasterSystemDiskCategory string    `json:"masterSystemDiskCategory,omitempty" yaml:"masterSystemDiskCategory,omitempty"`
	MasterSystemDiskSize     int       `json:"masterSystemDiskSize,omitempty" yaml:"masterSystemDiskSize,omitempty"`
	KeyPair                  string    `json:"keyPair,omitempty" yaml:"keyPair,omitempty"`
	NodePools                NodePools `json:"nodePools,omitempty" yaml:"nodePools,omitempty"`
}

// UpdateClusterACSK describes Alibaba's node fields of an UpdateCluster request
type UpdateClusterACSK struct {
	NodePools NodePools `json:"nodePools,omitempty"`
}

// AddDefaults puts default values to optional field(s)
func (c *CreateClusterACSK) AddDefaults() error {
	if c.MasterInstanceType == "" {
		c.MasterInstanceType = DefaultMasterInstanceType
	}
	if c.MasterSystemDiskCategory == "" {
		c.MasterSystemDiskCategory = DefaultMasterSystemDiskCategory
	}
	if c.MasterSystemDiskSize < DefaultMasterSystemDiskSize {
		c.MasterSystemDiskSize = DefaultMasterSystemDiskSize
	}

	if len(c.NodePools) == 0 {
		return pkgErrors.ErrorAlibabaNodePoolFieldIsEmpty
	}
	for i, np := range c.NodePools {
		if np.InstanceType == "" {
			c.NodePools[i].InstanceType = DefaultWorkerInstanceType
		}
		//if np.SystemDiskCategory == "" {
		//	c.NodePools[i].SystemDiskCategory = DefaultWorkerSystemDiskCategory
		//}
		//if np.SystemDiskSize < DefaultWorkerSystemDiskSize {
		//	c.NodePools[i].SystemDiskSize = DefaultWorkerSystemDiskSize
		//}
	}

	return nil
}

func ValidateNodePools(nps NodePools) error {
	if len(nps) == 0 {
		return pkgErrors.ErrorAlibabaNodePoolFieldIsEmpty
	}

	// Alibaba only supports 20 nodepools
	if len(nps) > 20 {
		return pkgErrors.ErrorAlibabaNodePoolFieldLenError
	}

	for _, np := range nps {
		if np.MinCount < 1 {
			return pkgErrors.ErrorAlibabaMinNumberOfNodes
		}
	}
	return nil
}

func (c *CreateClusterACSK) Validate() error {
	if c == nil {
		return pkgErrors.ErrorAlibabaFieldIsEmpty
	}
	if c.RegionID == "" {
		return pkgErrors.ErrorAlibabaRegionIDFieldIsEmpty
	}
	if c.ZoneID == "" {
		return pkgErrors.ErrorAlibabaZoneIDFieldIsEmpty
	}
	if c.NodePools == nil {
		return pkgErrors.ErrorAlibabaNodePoolFieldIsEmpty
	}

	return ValidateNodePools(c.NodePools)
}

func (c *UpdateClusterACSK) Validate() error {
	if c == nil {
		return pkgErrors.ErrorAlibabaFieldIsEmpty
	}

	return ValidateNodePools(c.NodePools)
}

// ClusterProfileACSK describes an Alibaba CS profile
type ClusterProfileACSK struct {
	RegionID  string               `json:"regionId"`
	ZoneID    string               `json:"zoneId"`
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}
