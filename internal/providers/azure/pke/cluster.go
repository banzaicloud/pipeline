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
	"encoding/json"

	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"

	"emperror.dev/errors"
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
	Max          uint
	Min          uint
	Name         string
	Roles        []string
	Subnet       Subnetwork
	Zones        []string
}

type AccessPoint struct {
	Name    string
	Address string
}

type AccessPoints []AccessPoint

func (a AccessPoints) Exists(name string) bool {
	for _, ap := range a {
		if ap.Name == name {
			return true
		}
	}
	return false
}

func (a AccessPoints) Get(name string) *AccessPoint {
	for i := range a {
		if a[i].Name == name {
			return &a[i]
		}
	}
	return nil
}

func (a AccessPoints) Marshal() (string, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return "", errors.WrapIf(err, "failed to marshall access point list")
	}

	return string(data), nil
}

func (a *AccessPoints) Unmarshal(data string) error {
	return errors.WrapIf(json.Unmarshal([]byte(data), a), "failed to unmarshal access point list")
}

type APIServerAccessPoint string

func (a APIServerAccessPoint) GetName() string {
	return string(a)
}

type APIServerAccessPoints []APIServerAccessPoint

func (a APIServerAccessPoints) Exists(name string) bool {
	for _, ap := range a {
		if ap.GetName() == name {
			return true
		}
	}
	return false
}

func (a APIServerAccessPoints) Marshal() (string, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return "", errors.WrapIf(err, "failed to marshall api server access point list")
	}

	return string(data), nil
}

// PKEOnAzureCluster defines fields for PKE-on-Azure clusters
type PKEOnAzureCluster struct {
	intCluster.ClusterBase

	Location         string
	NodePools        []NodePool
	ResourceGroup    ResourceGroup
	VirtualNetwork   VirtualNetwork
	Kubernetes       intPKE.Kubernetes
	ActiveWorkflowID string
	HTTPProxy        intPKE.HTTPProxy

	Monitoring   bool
	Logging      bool
	SecurityScan bool
	TtlMinutes   uint

	AccessPoints          AccessPoints
	APIServerAccessPoints APIServerAccessPoints
}

func (c PKEOnAzureCluster) HasActiveWorkflow() bool {
	return c.ActiveWorkflowID != ""
}

func GetVMSSName(clusterName, nodePoolName string) string {
	return clusterName + "-" + nodePoolName
}

func GetRouteTableName(clusterName string) string {
	return clusterName + "-route-table"
}

func GetBackendAddressPoolName() string {
	return "backend-address-pool"
}

func GetOutboundBackendAddressPoolName() string {
	return "outbound-backend-address-pool"
}

func GetInboundNATPoolName() string {
	return "ssh-inbound-nat-pool"
}

func GetLoadBalancerName(clusterName string) string {
	return clusterName // LB name must match the value passed to pke install master --kubernetes-cluster-name
}

func GetPublicIPAddressName(clusterName string) string {
	return clusterName + "-pip-in"
}

func GetFrontEndIPConfigName() string {
	return "frontend-ip-config"
}

func GetApiServerLBRuleName() string {
	return "api-server-rule"
}
