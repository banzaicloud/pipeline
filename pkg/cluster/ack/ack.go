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

package ack

import (
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// NodePool describes Alibaba's node fields of a CreateCluster/Update request
type NodePool struct {
	InstanceType string            `json:"instanceType"`
	MinCount     int               `json:"minCount"`
	MaxCount     int               `json:"maxCount"`
	Labels       map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type NodePools map[string]*NodePool

// CreateClusterACK
type CreateClusterACK struct {
	RegionID                 string    `json:"regionId" yaml:"regionId"`
	ZoneID                   string    `json:"zoneId" yaml:"zoneId"`
	MasterInstanceType       string    `json:"masterInstanceType,omitempty" yaml:"masterInstanceType,omitempty"`
	MasterSystemDiskCategory string    `json:"masterSystemDiskCategory,omitempty" yaml:"masterSystemDiskCategory,omitempty"`
	MasterSystemDiskSize     int       `json:"masterSystemDiskSize,omitempty" yaml:"masterSystemDiskSize,omitempty"`
	KeyPair                  string    `json:"keyPair,omitempty" yaml:"keyPair,omitempty"`
	NodePools                NodePools `json:"nodePools,omitempty" yaml:"nodePools,omitempty"`
	VSwitchID                string    `json:"vswitchId,omitempty" yaml:"vswitchId,omitempty"`
}

// UpdateClusterACK describes Alibaba's node fields of an UpdateCluster request
type UpdateClusterACK struct {
	NodePools NodePools `json:"nodePools,omitempty"`
}

// AddDefaults puts default values to optional field(s)
func (c *CreateClusterACK) AddDefaults() error {
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
	}

	return nil
}

func ValidateNodePools(nps NodePools) error {
	if len(nps) == 0 {
		return pkgErrors.ErrorAlibabaNodePoolFieldIsEmpty
	}

	if len(nps) > AlibabaMaxNodePoolSize {
		return pkgErrors.ErrorAlibabaNodePoolFieldLenError
	}

	for _, np := range nps {
		if np.MinCount < 1 {
			return pkgErrors.ErrorAlibabaMinNumberOfNodes
		}
		if np.MaxCount < np.MinCount && np.MaxCount > 1000 {
			return pkgErrors.ErrorAlibabaMaxNumberOfNodes
		}
	}
	return nil
}

func (c *CreateClusterACK) Validate() error {
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

func (c *UpdateClusterACK) Validate() error {
	if c == nil {
		return pkgErrors.ErrorAlibabaFieldIsEmpty
	}

	return ValidateNodePools(c.NodePools)
}

// ClusterProfileACK describes an Alibaba CS profile
type ClusterProfileACK struct {
	RegionID  string               `json:"regionId"`
	ZoneID    string               `json:"zoneId"`
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}
