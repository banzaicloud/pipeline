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

package banzaicloud

// CreateClusterBanzaiCloud describes Pipeline's EC2/BanzaiCloud fields of a CreateCluster request
type CreateClusterBanzaiCloud struct {
	Network    Network    `json:"network,omitempty" yaml:"network,omitempty"`
	NodePools  NodePools  `json:"nodepools,omitempty" yaml:"nodepools,omitempty"`
	Kubernetes Kubernetes `json:"kubernetes,omitempty" yaml:"kubernetes,omitempty"`
	KubeADM    KubeADM    `json:"kubeadm,omitempty" yaml:"kubeadm,omitempty"`
	CRI        CRI        `json:"cri,omitempty" yaml:"cri,omitempty"`
}

type Network struct {
	ServiceCIDR      string          `json:"serviceCIDR" yaml:"serviceCIDR"`
	PodCIDR          string          `json:"podCIDR" yaml:"podCIDR"`
	Provider         NetworkProvider `json:"provider" yaml:"provider"`
	APIServerAddress string          `json:"apiServerAddress" yaml:"apiServerAddress"`
}

type NetworkProvider string

const (
	NPWeave NetworkProvider = "weave"
)

type NodePools []NodePool

type NodePool struct {
	Name           string                 `json:"name" yaml:"name"`
	Roles          Roles                  `json:"roles" yaml:"roles"`
	Hosts          Hosts                  `json:"hosts" yaml:"hosts"`
	Provider       NodePoolProvider       `json:"provider" yaml:"provider"`
	ProviderConfig map[string]interface{} `json:"providerConfig" yaml:"providerConfig"`
}

type NodePoolProvider string

const (
	NPPAmazon NodePoolProvider = "amazon"
)

type Roles []Role
type Role string

const (
	RoleMaster         Role = "master"
	RoleWorker         Role = "worker"
	RolePipelineSystem Role = "pipeline-system"
)

type Hosts []Host
type Host struct {
	Name             string `json:"name" yaml:"name"`
	PrivateIP        string `json:"privateIP" yaml:"privateIP"`
	NetworkInterface string `json:"networkInterface" yaml:"networkInterface"`
	Roles            Roles  `json:"roles" yaml:"roles"`
	Labels           Labels `json:"labels" yaml:"labels"`
	Taints           Taints `json:"taints" yaml:"taints"`
}

type Labels map[string]string

type Taints []Taint
type Taint string

type AmazonProviderConfig struct {
	AutoScalingGroup struct {
		Name                    string  `json:"name" yaml:"name"`
		Image                   string  `json:"image" yaml:"image"`
		Zones                   Zones   `json:"zones" yaml:"zones"`
		InstanceType            string  `json:"instanceType" yaml:"instanceType"`
		LaunchConfigurationName string  `json:"launchConfigurationName" yaml:"launchConfigurationName"`
		LaunchTemplate          string  `json:"launchTemplate" yaml:"launchTemplate"`
		VPCID                   string  `json:"vpcID" yaml:"vpcID"`
		SecurityGroupID         string  `json:"securityGroupID" yaml:"securityGroupID"`
		Subnets                 Subnets `json:"subnets" yaml:"subnets"`
		Tags                    Tags    `json:"tags" yaml:"tags"`
		Size                    struct {
			Min int `json:"min" yaml:"min"`
			Max int `json:"max" yaml:"max"`
		} `json:"size" yaml:"size"`
	} `json:"autoScalingGroup" yaml:"autoScalingGroup"`
}

type Zones []Zone
type Zone string

type Subnets []Subnet
type Subnet string

type Tags map[string]string

type Kubernetes struct {
	Version string `json:"version" yaml:"version"`
	RBAC    RBAC   `json:"rbac" yaml:"rbac"`
}

type RBAC struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

type CRI struct {
	Runtime       Runtime                `json:"runtime" yaml:"runtime"`
	RuntimeConfig map[string]interface{} `json:"runtimeConfig" yaml:"runtimeConfig"`
}
type Runtime string

const (
	CRIDocker     Runtime = "docker"
	CRIContainerd Runtime = "containerd"
)

type KubeADM struct {
	ExtraArgs ExtraArgs `json:"extraArgs" yaml:"extraArgs"`
}

type ExtraArgs []ExtraArg
type ExtraArg string
