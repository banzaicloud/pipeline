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
	"context"

	"github.com/Azure/go-autorest/autorest/to"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/goph/emperror"
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

	Location         string
	NodePools        []NodePool
	ResourceGroup    ResourceGroup
	VirtualNetwork   VirtualNetwork
	Kubernetes       intPKE.Kubernetes
	ActiveWorkflowID string

	Monitoring   bool
	Logging      bool
	ServiceMesh  bool
	SecurityScan bool
	TtlMinutes   uint
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

func GetBackendAddressPoolIDsForCluster(ctx context.Context, client azure.LoadBalancersClient, cluster PKEOnAzureCluster) (bapIDs map[string]string, err error) {
	lb, err := client.Get(ctx, cluster.ResourceGroup.Name, GetLoadBalancerName(cluster.Name), "")
	if err = emperror.Wrap(err, "failed to get load balancer"); err != nil {
		return
	}
	if lb.BackendAddressPools == nil {
		return
	}
	bapIDs = make(map[string]string, len(*lb.BackendAddressPools))
	for _, bap := range *lb.BackendAddressPools {
		bapIDs[to.String(bap.Name)] = to.String(bap.ID)
	}
	return
}

func GetInboundNATPoolIDsForCluster(ctx context.Context, client azure.LoadBalancersClient, cluster PKEOnAzureCluster) (inpIDs map[string]string, err error) {
	lb, err := client.Get(ctx, cluster.ResourceGroup.Name, GetLoadBalancerName(cluster.Name), "")
	if err = emperror.Wrap(err, "failed to get load balancer"); err != nil {
		return
	}
	if lb.InboundNatPools == nil {
		return
	}
	inpIDs = make(map[string]string, len(*lb.InboundNatPools))
	for _, inp := range *lb.InboundNatPools {
		inpIDs[to.String(inp.Name)] = to.String(inp.ID)
	}
	return
}

func GetPublicIPAddressForCluster(ctx context.Context, client azure.PublicIPAddressesClient, cluster PKEOnAzureCluster) (string, error) {
	pip, err := client.Get(ctx, cluster.ResourceGroup.Name, GetPublicIPAddressName(cluster.Name), "")
	return to.String(pip.ID), emperror.Wrap(err, "failed to get public IP address")
}

func GetRouteTableIDForCluster(ctx context.Context, client azure.RouteTablesClient, cluster PKEOnAzureCluster) (string, error) {
	rt, err := client.Get(ctx, cluster.ResourceGroup.Name, GetRouteTableName(cluster.Name), "")
	return to.String(rt.ID), emperror.Wrap(err, "failed to get route table")
}

func GetSecurityGroupIDsForCluster(ctx context.Context, client azure.SecurityGroupsClient, cluster PKEOnAzureCluster) (sgIDs map[string]string, err error) {
	resPage, err := client.List(ctx, cluster.ResourceGroup.Name)
	if err = emperror.Wrap(err, "failed to list security groups"); err != nil {
		return
	}
	sgIDs = make(map[string]string)
	for {
		for _, sg := range resPage.Values() {
			sgIDs[to.String(sg.Name)] = to.String(sg.ID)
		}
		if !resPage.NotDone() {
			return
		}
		if err = emperror.Wrap(resPage.NextWithContext(ctx), "failed to advance security group list result page"); err != nil {
			return
		}
	}
}

func GetSubnetIDsForCluster(ctx context.Context, client azure.SubnetsClient, cluster PKEOnAzureCluster) (subnetIDs map[string]string, err error) {
	resPage, err := client.List(ctx, cluster.ResourceGroup.Name, cluster.VirtualNetwork.Name)
	if err = emperror.Wrap(err, "failed to list subnets"); err != nil {
		return
	}
	subnetIDs = make(map[string]string)
	for {
		for _, subnet := range resPage.Values() {
			subnetIDs[to.String(subnet.Name)] = to.String(subnet.ID)
		}
		if !resPage.NotDone() {
			return
		}
		if err = emperror.Wrap(resPage.NextWithContext(ctx), "failed to advance subnet list result page"); err != nil {
			return
		}
	}
}
