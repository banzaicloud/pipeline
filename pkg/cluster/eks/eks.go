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

package eks

import (
	"github.com/Masterminds/semver"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/goph/emperror"
)

// CreateClusterEKS describes Pipeline's Amazon EKS fields of a CreateCluster request
type CreateClusterEKS struct {
	Version      string               `json:"version,omitempty" yaml:"version,omitempty"`
	NodePools    map[string]*NodePool `json:"nodePools,omitempty" yaml:"nodePools,omitempty"`
	Vpc          *ClusterVPC          `json:"vpc,omitempty" yaml:"vpc,omitempty"`
	RouteTableId string               `json:"routeTableId,omitempty" yaml:"routeTableId,omitempty"`
	Subnets      []*ClusterSubnet     `json:"subnets,omitempty" yaml:"subnets,omitempty"`
}

// UpdateClusterAmazonEKS describes Amazon EKS's node fields of an UpdateCluster request
type UpdateClusterAmazonEKS struct {
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}

// NodePool describes Amazon's node fields of a CreateCluster/Update request
type NodePool struct {
	InstanceType string            `json:"instanceType" yaml:"instanceType"`
	SpotPrice    string            `json:"spotPrice" yaml:"spotPrice"`
	Autoscaling  bool              `json:"autoscaling" yaml:"autoscaling"`
	MinCount     int               `json:"minCount" yaml:"minCount"`
	MaxCount     int               `json:"maxCount" yaml:"maxCount"`
	Count        int               `json:"count" yaml:"count"`
	Image        string            `json:"image" yaml:"image"`
	Labels       map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// ClusterVPC describes the VPC for creating an EKS cluster
type ClusterVPC struct {
	VpcId string `json:"vpcId,omitempty" yaml:"vpcId,omitempty"`
	Cidr  string `json:"cidr,omitempty" yaml:"cidr,omitempty"`
}

// ClusterSubnet describes a subnet for EKS cluster
type ClusterSubnet struct {
	SubnetId string `json:"subnetId,omitempty" yaml:"subnetId,omitempty"`
	Cidr     string `json:"cidr,omitemnpty" yaml:"cidr,omitempty"`
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

	// --- [Label validation]--- //
	if err := pkgCommon.ValidateNodePoolLabels(a.Labels); err != nil {
		return err
	}

	return nil
}

// ValidateForUpdate checks Amazon's node fields
func (a *NodePool) ValidateForUpdate() error {

	// ---- [ Min & Max count fields are required in case of autoscaling ] ---- //
	if a.Autoscaling {
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

	// --- [Label validation]--- //
	if err := pkgCommon.ValidateNodePoolLabels(a.Labels); err != nil {
		return err
	}

	return nil
}

// Validate validates Amazon EKS cluster create request
func (eks *CreateClusterEKS) Validate() error {
	if eks == nil {
		return pkgErrors.ErrorAmazonEksFieldIsEmpty
	}

	// validate K8s version
	isValid, err := isValidVersion(eks.Version)
	if err != nil {
		return emperror.Wrap(err, "couldn't validate Kubernetes version")
	}
	if !isValid {
		return pkgErrors.ErrorNotValidKubernetesVersion
	}

	for _, np := range eks.NodePools {
		if err := np.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// AddDefaults puts default values to optional field(s)
func (eks *CreateClusterEKS) AddDefaults(location string) error {
	if eks == nil {
		return pkgErrors.ErrorAmazonEksFieldIsEmpty
	}

	defaultImage, err := GetDefaultImageID(location, eks.Version)
	if err != nil {
		return emperror.Wrapf(err, "couldn't get EKS AMI for Kubernetes version %q in region %q", eks.Version, location)
	}

	if len(eks.NodePools) == 0 {
		return pkgErrors.ErrorAmazonEksNodePoolFieldIsEmpty
	}

	for i, np := range eks.NodePools {
		if len(np.Image) == 0 {
			eks.NodePools[i].Image = defaultImage
		}
	}

	if eks.Vpc == nil {
		eks.Vpc = &ClusterVPC{
			Cidr: "192.168.0.0/16",
		}
	}

	if len(eks.Subnets) == 0 {
		eks.Subnets = append(eks.Subnets,
			&ClusterSubnet{
				Cidr: "192.168.64.0/20",
			},
			&ClusterSubnet{
				Cidr: "192.168.80.0/20",
			},
		)
	}

	return nil
}

// Validate validates the update request (only EKS part). If any of the fields is missing, the method fills
// with stored data.
func (eks *UpdateClusterAmazonEKS) Validate() error {

	// ---- [ Amazon EKS field check ] ---- //
	if eks == nil {
		return pkgErrors.ErrorAmazonEksFieldIsEmpty
	}

	for _, np := range eks.NodePools {
		if err := np.ValidateForUpdate(); err != nil {
			return err
		}
	}

	return nil
}

// isValidVersion validates the given K8S version
func isValidVersion(version string) (bool, error) {
	constraint, err := semver.NewConstraint(">= 1.10, < 1.15")
	if err != nil {
		return false, emperror.Wrap(err, "couldn't create semver Kubernetes version check constraint")
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return false, emperror.Wrap(err, "couldn't create semver")
	}

	// TODO check if there is an AWS API that can tell us supported Kubernetes versions
	return constraint.Check(v), nil

}

// CertificateAuthority is a helper struct for AWS kube config JSON parsing
type CertificateAuthority struct {
	Data string `json:"data,omitempty"`
}

// ClusterProfileEKS describes an Amazon EKS profile
type ClusterProfileEKS struct {
	Version   string               `json:"version,omitempty"`
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}

// CreateAmazonEksObjectStoreBucketProperties describes the properties of
// S3 bucket creation request
type CreateAmazonEksObjectStoreBucketProperties struct {
	Location string `json:"location" binding:"required"`
}
