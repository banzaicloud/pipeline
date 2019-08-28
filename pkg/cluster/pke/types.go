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

package pke

import "github.com/pkg/errors"

// TODO add required field to KubeADM if applicable

// CreateClusterPKE describes Pipeline's EC2/BanzaiCloud fields of a CreateCluster request
type CreateClusterPKE struct {
	Network    Network    `json:"network,omitempty" yaml:"network,omitempty" binding:"required"`
	NodePools  NodePools  `json:"nodepools,omitempty" yaml:"nodepools,omitempty" binding:"required"`
	Kubernetes Kubernetes `json:"kubernetes,omitempty" yaml:"kubernetes,omitempty" binding:"required"`
	KubeADM    KubeADM    `json:"kubeadm,omitempty" yaml:"kubeadm,omitempty"`
	CRI        CRI        `json:"cri,omitempty" yaml:"cri,omitempty" binding:"required"`
}

// UpdateClusterPKE describes Pipeline's EC2/BanzaiCloud fields of a UpdateCluster request
type UpdateClusterPKE struct {
	NodePools UpdateNodePools `json:"nodepools,omitempty" yaml:"nodepools,omitempty" binding:"required"`
}

func (a *UpdateClusterPKE) Validate() error {
	// TODO implement
	return nil
}

type UpdateNodePools map[string]UpdateNodePool

type UpdateNodePool struct {
	InstanceType string  `json:"instanceType" yaml:"instanceType"`
	SpotPrice    string  `json:"spotPrice" yaml:"spotPrice"`
	Autoscaling  bool    `json:"autoscaling" yaml:"autoscaling"`
	MinCount     int     `json:"minCount" yaml:"minCount"`
	MaxCount     int     `json:"maxCount" yaml:"maxCount"`
	Count        int     `json:"count" yaml:"count"`
	Subnets      Subnets `json:"subnets,omitempty" yaml:"subnets,omitempty"`
}

type Network struct {
	ServiceCIDR      string                 `json:"serviceCIDR" yaml:"serviceCIDR"`
	PodCIDR          string                 `json:"podCIDR" yaml:"podCIDR"`
	Provider         NetworkProvider        `json:"provider" yaml:"provider"`
	APIServerAddress string                 `json:"apiServerAddress" yaml:"apiServerAddress"`
	ProviderConfig   map[string]interface{} `json:"cloudProviderConfig" yaml:"cloudProviderConfig"`
}

type NetworkProvider string

const (
	NPWeave NetworkProvider = "weave"
)

type NodePools []NodePool

type NodePool struct {
	Name           string                 `json:"name" yaml:"name" binding:"required"`
	Roles          Roles                  `json:"roles" yaml:"roles" binding:"required"`
	Hosts          Hosts                  `json:"hosts" yaml:"hosts"`
	Provider       NodePoolProvider       `json:"provider" yaml:"provider" binding:"required"`
	ProviderConfig map[string]interface{} `json:"providerConfig" yaml:"providerConfig" binding:"required"`
	Labels         map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	Autoscaling    bool                   `json:"autoscaling" yaml:"autoscaling"`
}

type NodePoolProvider string

const (
	NPPAmazon NodePoolProvider = "amazon"
)

type Roles []Role
type Role string

const (
	RoleMaster               Role   = "master"
	RoleWorker               Role   = "worker"
	RolePipelineSystem       Role   = "pipeline-system"
	TaintKeyMaster           string = "node-role.kubernetes.io/master"
	NodeLabelKeyMasterWorker string = "node-role.kubernetes.io/master-worker"
)

type Hosts []Host
type Host struct {
	Name             string `json:"name" yaml:"name" binding:"required"`
	PrivateIP        string `json:"privateIP" yaml:"privateIP" binding:"required"`
	NetworkInterface string `json:"networkInterface" yaml:"networkInterface" binding:"required"`
	Roles            Roles  `json:"roles" yaml:"roles" binding:"required"`
	Labels           Labels `json:"labels" yaml:"labels" binding:"required"`
	Taints           Taints `json:"taints" yaml:"taints" binding:"required"`
}

type Labels map[string]string

type Taints []Taint
type Taint string

// TODO add required field to LaunchTemplate if applicable
type AmazonProviderConfig struct {
	AutoScalingGroup struct {
		Name                    string  `json:"name" yaml:"name" binding:"required"`
		Image                   string  `json:"image" yaml:"image" binding:"required"`
		Zones                   Zones   `json:"zones" yaml:"zones" binding:"required"`
		InstanceType            string  `json:"instanceType" yaml:"instanceType" binding:"required"`
		LaunchConfigurationName string  `json:"launchConfigurationName" yaml:"launchConfigurationName" binding:"required"`
		LaunchTemplate          string  `json:"launchTemplate" yaml:"launchTemplate"`
		VPCID                   string  `json:"vpcID" yaml:"vpcID" binding:"required"`
		SecurityGroupID         string  `json:"securityGroupID" yaml:"securityGroupID" binding:"required"`
		Subnets                 Subnets `json:"subnets" yaml:"subnets" binding:"required"`
		Tags                    Tags    `json:"tags" yaml:"tags" binding:"required"`
		Size                    struct {
			Desired int `json:"desired" yaml:"desired"`
			Min     int `json:"min" yaml:"min" binding:"required"`
			Max     int `json:"max" yaml:"max" binding:"required"`
		} `json:"size" yaml:"size" binding:"required"`
	} `json:"autoScalingGroup" yaml:"autoScalingGroup" binding:"required"`
}

// AddDefaults puts default values to optional field(s)
func (pke *CreateClusterPKE) AddDefaults() error {
	if pke == nil {
		return errors.New("Required field 'pke' is empty.")
	}

	if pke.Network.PodCIDR == "" {
		pke.Network.PodCIDR = "10.200.0.0/16"
	}
	if pke.Network.ServiceCIDR == "" {
		pke.Network.ServiceCIDR = "10.32.0.0/24"
	}
	if pke.Network.Provider == "" {
		pke.Network.Provider = NPWeave
	}

	return nil
}

type Zones []Zone
type Zone string

type Subnets []Subnet
type Subnet string

type Tags map[string]string

type Kubernetes struct {
	Version string `json:"version" yaml:"version" binding:"required"`
	RBAC    RBAC   `json:"rbac" yaml:"rbac" binding:"required"`
	OIDC    OIDC   `json:"oidc" yaml:"oidc"`
}

type RBAC struct {
	Enabled bool `json:"enabled" yaml:"enabled" binding:"required"`
}

type OIDC struct {
	Enabled bool `json:"enabled" yaml:"enabled" binding:"required"`
}

// TODO add required field to RuntimeConfig if applicable
type CRI struct {
	Runtime       Runtime                `json:"runtime" yaml:"runtime" binding:"required"`
	RuntimeConfig map[string]interface{} `json:"runtimeConfig" yaml:"runtimeConfig"`
}
type Runtime string

const (
	CRIDocker     Runtime = "docker"
	CRIContainerd Runtime = "containerd"
)

// //TODO add required field to ExtraArgs if applicable
type KubeADM struct {
	ExtraArgs ExtraArgs `json:"extraArgs" yaml:"extraArgs"`
}

type ExtraArgs []ExtraArg
type ExtraArg string
