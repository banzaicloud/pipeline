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

package ekscluster

import (
	"context"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/Masterminds/semver/v3"
	"github.com/ghodss/yaml"

	eks2 "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/global/globaleks"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// CreateClusterEKS describes Pipeline's Amazon EKS fields of a CreateCluster request
type CreateClusterEKS struct {
	Version          string               `json:"version,omitempty" yaml:"version,omitempty"`
	EncryptionConfig []EncryptionConfig   `json:"encryptionConfig,omitempty" yaml:"encryptionConfig,omitempty"`
	NodePools        map[string]*NodePool `json:"nodePools,omitempty" yaml:"nodePools,omitempty"`
	Vpc              *ClusterVPC          `json:"vpc,omitempty" yaml:"vpc,omitempty"`
	RouteTableId     string               `json:"routeTableId,omitempty" yaml:"routeTableId,omitempty"`
	// Subnets for EKS master and worker nodes. All worker nodes will be launched in the same subnet
	// (the first subnet in the list - which may not coincide with first subnet in the cluster create request payload as
	// the deserialization may change the order) unless a subnet is specified for the workers that belong to a node pool at node pool level.
	Subnets  []*Subnet  `json:"subnets,omitempty" yaml:"subnets,omitempty"`
	IAM      ClusterIAM `json:"iam,omitempty" yaml:"iam,omitempty"`
	LogTypes []string   `json:"logTypes,omitempty" yaml:"logTypes,omitempty"`

	// List of access point references for the API server; currently, public and private are the only valid values.
	// Default: ["public"]
	APIServerAccessPoints []string          `json:"apiServerAccessPoints,omitempty" yaml:"apiServerAccessPoints,omitempty"`
	Tags                  map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	AuthConfig            *AuthConfig       `json:"authConfig,omitempty" yaml:"authConfig,omitempty"`
}

func (cm *AuthConfig) ConvertToString() (string, error) {
	if cm == nil {
		return "", nil
	}

	mapRoles, err := yaml.Marshal(cm.MapRoles)
	if err != nil {
		return "", errors.WrapIf(err, "failed to marshal map roles")
	}

	mapUsers, err := yaml.Marshal(cm.MapUsers)
	if err != nil {
		return "", errors.WrapIf(err, "failed to marshal map users")
	}

	mapAccounts, err := yaml.Marshal(cm.MapAccounts)
	if err != nil {
		return "", errors.WrapIf(err, "failed to marshal map accounts")
	}

	mrIndent := strings.ReplaceAll(string(mapRoles), "\n", "\n  ")
	muIndent := strings.ReplaceAll(string(mapUsers), "\n", "\n  ")
	maIndent := strings.ReplaceAll(string(mapAccounts), "\n", "\n  ")

	return fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
 name: aws-auth
 namespace: kube-system
data:
 mapRoles: |
  %s
 mapUsers: |
  %s
 mapAccounts: |
  %s`, mrIndent, muIndent, maIndent), nil
}

type AuthConfig struct {
	MapRoles    []MapRoles `json:"mapRoles,omitempty" yaml:"mapRoles,omitempty"`
	MapUsers    []MapUsers `json:"mapUsers,omitempty" yaml:"mapUsers,omitempty"`
	MapAccounts []string   `json:"mapAccounts,omitempty" yaml:"mapAccounts,omitempty"`
}

type MapRoles struct {
	Groups   []string `json:"groups,omitempty" yaml:"groups,omitempty"`
	RoleARN  string   `json:"rolearn,omitempty" yaml:"rolearn,omitempty"`
	Username string   `json:"username,omitempty" yaml:"username,omitempty"`
}

type MapUsers struct {
	Groups   []string `json:"groups,omitempty" yaml:"groups,omitempty"`
	UserARN  string   `json:"userarn,omitempty" yaml:"userarn,omitempty"`
	Username string   `json:"username,omitempty" yaml:"username,omitempty"`
}

// UpdateClusterAmazonEKS describes Amazon EKS's node fields of an UpdateCluster request
type UpdateClusterAmazonEKS struct {
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}

// NodePool describes Amazon's node fields of a CreateCluster/Update request
type NodePool struct {
	InstanceType     string                    `json:"instanceType" yaml:"instanceType"`
	SpotPrice        string                    `json:"spotPrice" yaml:"spotPrice"`
	Autoscaling      bool                      `json:"autoscaling" yaml:"autoscaling"`
	MinCount         int                       `json:"minCount" yaml:"minCount"`
	MaxCount         int                       `json:"maxCount" yaml:"maxCount"`
	Count            int                       `json:"count" yaml:"count"`
	VolumeEncryption *NodePoolVolumeEncryption `json:"volumeEncryption,omitempty" yaml:"volumeEncryption,omitempty"`
	VolumeSize       int                       `json:"volumeSize" yaml:"volumeSize"`
	Image            string                    `json:"image" yaml:"image"`
	Labels           map[string]string         `json:"labels,omitempty" yaml:"labels,omitempty"`

	// SecurityGroups collects the user provided node security group IDs for the
	// node pool.
	SecurityGroups   []string `json:"securityGroups,omitempty" yaml:"securityGroups,omitempty"`
	UseInstanceStore *bool    `json:"useInstanceStore,omitempty" yaml:"useInstanceStore,omitempty"`

	// Subnet for worker nodes of this node pool. If not specified than worker nodes
	// are launched in the same subnet in one of the subnets from the list of subnets of the EKS cluster
	Subnet *Subnet `json:"subnet,omitempty" yaml:"subnet,omitempty"`
}

// NodePoolVolumeEncryption describes the EKS node pool encryption details.
type NodePoolVolumeEncryption struct {
	Enabled          bool   `json:"enabled" yaml:"enabled"`
	EncryptionKeyARN string `json:"encryptionKeyARN" yaml:"encryptionKeyARN"`
}

// ClusterIAM describes the IAM config for creating an EKS cluster
type ClusterIAM struct {
	ClusterRoleID      string `json:"clusterRoleId,omitempty" yaml:"clusterRoleId,omitempty"`
	NodeInstanceRoleID string `json:"nodeInstanceRoleId,omitempty" yaml:"nodeInstanceRoleId,omitempty"`
	// marks if the userid associated with the clusters aws secret has to be used in kubeconfig (bypasses user creation)
	DefaultUser bool `json:"defaultUser,omitempty" yaml:"defaultUser,omitempty"`
}

// ClusterVPC describes the VPC for creating an EKS cluster
type ClusterVPC struct {
	VpcId string `json:"vpcId,omitempty" yaml:"vpcId,omitempty"`
	Cidr  string `json:"cidr,omitempty" yaml:"cidr,omitempty"`
}

type EncryptionConfig struct {
	Provider  Provider
	Resources []string
}

type Provider struct {
	KeyARN string
}

// Subnet describes a subnet for EKS cluster
type Subnet struct {
	// Id of existing subnet to use for creating the EKS cluster. If not provided new subnet will be created.
	SubnetId string `json:"subnetId,omitempty" yaml:"subnetId,omitempty"`
	// The CIDR range for the subnet in case new Subnet is created.
	Cidr string `json:"cidr,omitempty" yaml:"cidr,omitempty"`
	// The AZ to create the subnet into.
	AvailabilityZone string `json:"availabilityZone,omitempty" yaml:"availabilityZone,omitempty"`
}

const (
	DEFAULT_VPC_CIDR     = "192.168.0.0/16"
	DEFAULT_SUBNET0_CIDR = "192.168.64.0/20"
	DEFAULT_SUBNET1_CIDR = "192.168.80.0/20"
)

// Validate checks Amazon's node fields
func (a *NodePool) Validate(npName string) error {
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
		a.SpotPrice = eks2.DefaultSpotPrice
	}

	// --- [Label validation]--- //
	if err := pkgCommon.ValidateNodePoolLabels(npName, a.Labels); err != nil {
		return err
	}

	return nil
}

// ValidateForUpdate checks Amazon's node fields
func (a *NodePool) ValidateForUpdate(npName string) error {
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
	if err := pkgCommon.ValidateNodePoolLabels(npName, a.Labels); err != nil {
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
		return errors.WrapIf(err, "couldn't validate Kubernetes version")
	}
	if !isValid {
		return pkgErrors.ErrorNotValidKubernetesVersion
	}

	// validate node pools
	var errs []error
	for npName, np := range eks.NodePools {
		if err := np.Validate(npName); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Combine(errs...); err != nil {
		return err
	}

	return nil
}

// AddDefaults puts default values to optional field(s)
func (eks *CreateClusterEKS) AddDefaults(location string) error {
	if eks == nil {
		return pkgErrors.ErrorAmazonEksFieldIsEmpty
	}

	if len(eks.NodePools) == 0 {
		return pkgErrors.ErrorAmazonEksNodePoolFieldIsEmpty
	}

	if eks.Vpc == nil {
		eks.Vpc = &ClusterVPC{
			Cidr: DEFAULT_VPC_CIDR,
		}
	}

	if len(eks.Subnets) == 0 {
		eks.Subnets = append(eks.Subnets,
			&Subnet{
				Cidr:             DEFAULT_SUBNET0_CIDR,
				AvailabilityZone: fmt.Sprintf("%sa", location),
			},
			&Subnet{
				Cidr:             DEFAULT_SUBNET1_CIDR,
				AvailabilityZone: fmt.Sprintf("%sb", location),
			},
		)
	}

	for i, np := range eks.NodePools {
		if np.Image == "" {
			criteria := eks2.ImageSelectionCriteria{
				Region:            location,
				InstanceType:      np.InstanceType,
				KubernetesVersion: eks.Version,
			}

			image, err := globaleks.ImageSelector().SelectImage(context.Background(), criteria)
			if err != nil {
				return err
			}

			eks.NodePools[i].Image = image
		}

		if np != nil && np.Subnet == nil {
			np.Subnet = eks.Subnets[0]
		}
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

	// validate node pools
	var errs []error
	for npName, np := range eks.NodePools {
		if err := np.ValidateForUpdate(npName); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Combine(errs...); err != nil {
		return err
	}

	return nil
}

// isValidVersion validates the given K8S version
func isValidVersion(version string) (bool, error) {
	constraint, err := semver.NewConstraint(">= 1.15, <= 1.19")
	if err != nil {
		return false, errors.WrapIf(err, "couldn't create semver Kubernetes version check constraint")
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return false, errors.WrapIf(err, "couldn't create semver")
	}

	// TODO check if there is an AWS API that can tell us supported Kubernetes versions
	return constraint.Check(v), nil
}

// CertificateAuthority is a helper struct for AWS kube config JSON parsing
type CertificateAuthority struct {
	Data string `json:"data,omitempty"`
}

// CreateAmazonEksObjectStoreBucketProperties describes the properties of
// S3 bucket creation request
type CreateAmazonEksObjectStoreBucketProperties struct {
	Location string `json:"location" binding:"required"`
}
