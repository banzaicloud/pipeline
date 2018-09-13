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

package ec2

import (
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// CreateClusterEC2 describes Pipeline's Amazon fields of a CreateCluster request
type CreateClusterEC2 struct {
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
	Master    *CreateAmazonMaster  `json:"master,omitempty"`
}

// CreateAmazonMaster describes Amazon's master fields of a CreateCluster request
type CreateAmazonMaster struct {
	InstanceType string `json:"instanceType"`
	Image        string `json:"image"`
}

// NodePool describes Amazon's node fields of a CreateCluster/Update request
type NodePool struct {
	InstanceType string `json:"instanceType"`
	SpotPrice    string `json:"spotPrice"`
	Autoscaling  bool   `json:"autoscaling"`
	MinCount     int    `json:"minCount"`
	MaxCount     int    `json:"maxCount"`
	Count        int    `json:"count"`
	Image        string `json:"image"`
}

// UpdateClusterAmazon describes Amazon's node fields of an UpdateCluster request
type UpdateClusterAmazon struct {
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}

// Validate checks Amazon's node fields
func (a *NodePool) Validate() error {
	// ---- [ Node instanceType check ] ---- //
	if len(a.InstanceType) == 0 {
		return pkgErrors.ErrorInstancetypeFieldIsEmpty
	}

	// ---- [ Node image check ] ---- //
	if len(a.Image) == 0 {
		return pkgErrors.ErrorAmazonImageFieldIsEmpty
	}

	// ---- [ Min & Max count fields are required in case of autoscaling ] ---- //
	if a.Autoscaling {

		if a.MinCount == 0 {
			return pkgErrors.ErrorMinFieldRequiredError
		}
		if a.MaxCount == 0 {
			return pkgErrors.ErrorMaxFieldRequiredError
		}

	} else {
		// ---- [ Node min count check ] ---- //
		if a.MinCount == 0 {
			a.MinCount = pkgCommon.DefaultNodeMinCount
		}

		// ---- [ Node max count check ] ---- //
		if a.MaxCount == 0 {
			a.MaxCount = pkgCommon.DefaultNodeMaxCount
		}
	}

	// ---- [ Node min count <= max count check ] ---- //
	if a.MaxCount < a.MinCount {
		return pkgErrors.ErrorNodePoolMinMaxFieldError
	}

	if a.Count == 0 {
		a.Count = a.MinCount
	} else {
		if a.Count < a.MinCount || a.Count > a.MaxCount {
			return pkgErrors.ErrorNodePoolCountFieldError
		}
	}

	// ---- [ Node spot price ] ---- //
	if len(a.SpotPrice) == 0 {
		a.SpotPrice = DefaultSpotPrice
	}

	return nil
}

// ValidateForUpdate checks Amazon's node fields
func (a *NodePool) ValidateForUpdate() error {

	// ---- [ Min & Max count fields are required in case of autoscaling ] ---- //
	if a.Autoscaling {

		if a.MinCount == 0 {
			return pkgErrors.ErrorMinFieldRequiredError
		}
		if a.MaxCount == 0 {
			return pkgErrors.ErrorMaxFieldRequiredError
		}

	} else {
		// ---- [ Node min count check ] ---- //
		if a.MinCount == 0 {
			a.MinCount = pkgCommon.DefaultNodeMinCount
		}

		// ---- [ Node max count check ] ---- //
		if a.MaxCount == 0 {
			a.MaxCount = pkgCommon.DefaultNodeMaxCount
		}
	}

	// ---- [ Node min count <= max count check ] ---- //
	if a.MaxCount < a.MinCount {
		return pkgErrors.ErrorNodePoolMinMaxFieldError
	}

	if a.Count == 0 {
		a.Count = a.MinCount
	} else {
		if a.Count < a.MinCount || a.Count > a.MaxCount {
			return pkgErrors.ErrorNodePoolCountFieldError
		}
	}

	return nil
}

// Validate validates Amazon cluster create request
func (amazon *CreateClusterEC2) Validate() error {
	if amazon == nil {
		return pkgErrors.ErrorAmazonFieldIsEmpty
	}
	if amazon.Master == nil {
		return pkgErrors.ErrorAmazonMasterFieldIsEmpty
	}
	if amazon.Master.Image == "" {
		return pkgErrors.ErrorAmazonImageFieldIsEmpty
	}

	if amazon.Master.InstanceType == "" {
		amazon.Master.InstanceType = DefaultInstanceType
	}

	if len(amazon.NodePools) == 0 {
		return pkgErrors.ErrorAmazonNodePoolFieldIsEmpty
	}

	for _, np := range amazon.NodePools {
		if err := np.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// AddDefaults puts default values to optional field(s)
func (amazon *CreateClusterEC2) AddDefaults(location string) error {

	if amazon == nil {
		return pkgErrors.ErrorAmazonFieldIsEmpty
	}

	defaultImage := DefaultImages[location]

	if amazon.Master == nil {
		amazon.Master = &CreateAmazonMaster{
			InstanceType: DefaultInstanceType,
			Image:        defaultImage,
		}
	} else {

		if len(amazon.Master.InstanceType) == 0 {
			amazon.Master.InstanceType = DefaultInstanceType
		}

		if len(amazon.Master.Image) == 0 {
			amazon.Master.Image = defaultImage
		}

	}

	if len(amazon.NodePools) == 0 {
		return pkgErrors.ErrorAmazonNodePoolFieldIsEmpty
	}

	for i, np := range amazon.NodePools {
		if len(np.Image) == 0 {
			amazon.NodePools[i].Image = defaultImage
		}
	}

	return nil
}

// Validate validates the update request (only ec2 part). If any of the fields is missing, the method fills
// with stored data.
func (a *UpdateClusterAmazon) Validate() error {

	// ---- [ Amazon field check ] ---- //
	if a == nil {
		return pkgErrors.ErrorAmazonFieldIsEmpty
	}

	if len(a.NodePools) == 0 {
		return pkgErrors.ErrorAmazonNodePoolFieldIsEmpty
	}

	for _, np := range a.NodePools {
		if err := np.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ClusterProfileEC2 describes an Amazon profile
type ClusterProfileEC2 struct {
	Master    *ProfileMaster       `json:"master,omitempty"`
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}

// ProfileMaster describes an Amazon profile's master fields
type ProfileMaster struct {
	InstanceType string `json:"instanceType"`
	Image        string `json:"image"`
}
