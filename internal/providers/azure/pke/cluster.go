// Copyright © 2019 Banzai Cloud
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

import (
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
)

const PKEOnAzure = "pke-on-azure"

type ResourceGroup struct {
	Name string
}

type VirtualNetwork struct {
	Location string
	Name     string
}

type Subnetwork struct {
	Name string
}

type NodePool struct {
	Autoscaling  bool
	CreatedBy    uint
	DesiredCount uint
	InstanceType string
	Labels       map[string]string
	Max          uint
	Min          uint
	Name         string
	Roles        []string
	Subnet       Subnetwork
	Zones        []string
}

// PKEOnAzureCluster defines fields for PKE-on-Azure clusters
type PKEOnAzureCluster struct {
	intCluster.ClusterBase

	Location       string
	NodePools      []NodePool
	ResourceGroup  ResourceGroup
	VirtualNetwork VirtualNetwork
}
