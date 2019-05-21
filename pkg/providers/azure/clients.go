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

package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2018-03-31/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/monitor/mgmt/2017-09-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
)

// getAuthorizationBaseClient returns a BaseClient instance of the authorization package
func (cc *CloudConnection) getAuthorizationBaseClient() *authorization.BaseClient {
	if cc.cache.authorizationBaseClient == nil {
		cc.cache.authorizationBaseClient = &authorization.BaseClient{
			Client:         cc.client,
			BaseURI:        cc.env.ResourceManagerEndpoint,
			SubscriptionID: cc.creds.SubscriptionID,
		}
	}
	return cc.cache.authorizationBaseClient
}

// getComputeBaseClient returns a BaseClient instance of the compute package
func (cc *CloudConnection) getComputeBaseClient() *compute.BaseClient {
	if cc.cache.computeBaseClient == nil {
		cc.cache.computeBaseClient = &compute.BaseClient{
			Client:         cc.client,
			BaseURI:        cc.env.ResourceManagerEndpoint,
			SubscriptionID: cc.creds.SubscriptionID,
		}
	}
	return cc.cache.computeBaseClient
}

// getContainerServiceBaseClient returns a BaseClient instance of the containerservice package
func (cc *CloudConnection) getContainerServiceBaseClient() *containerservice.BaseClient {
	if cc.cache.containerServiceBaseClient == nil {
		cc.cache.containerServiceBaseClient = &containerservice.BaseClient{
			Client:         cc.client,
			BaseURI:        cc.env.ResourceManagerEndpoint,
			SubscriptionID: cc.creds.SubscriptionID,
		}
	}
	return cc.cache.containerServiceBaseClient
}

// getInsightsBaseClient returns a BaseClient instance of the insights package
func (cc *CloudConnection) getInsightsBaseClient() *insights.BaseClient {
	if cc.cache.insightsBaseClient == nil {
		cc.cache.insightsBaseClient = &insights.BaseClient{
			Client:         cc.client,
			BaseURI:        cc.env.ResourceManagerEndpoint,
			SubscriptionID: cc.creds.SubscriptionID,
		}
	}
	return cc.cache.insightsBaseClient
}

// getNetwokBaseClient returns a BaseClient instance of the network package
func (cc *CloudConnection) getNetworkBaseClient() *network.BaseClient {
	if cc.cache.networkBaseClient == nil {
		cc.cache.networkBaseClient = &network.BaseClient{
			Client:         cc.client,
			BaseURI:        cc.env.ResourceManagerEndpoint,
			SubscriptionID: cc.creds.SubscriptionID,
		}
	}
	return cc.cache.networkBaseClient
}

// getResourcesBaseClient returns a BaseClient instance of the resources package
func (cc *CloudConnection) getResourcesBaseClient() *resources.BaseClient {
	if cc.cache.resourcesBaseClient == nil {
		cc.cache.resourcesBaseClient = &resources.BaseClient{
			Client:         cc.client,
			BaseURI:        cc.env.ResourceManagerEndpoint,
			SubscriptionID: cc.creds.SubscriptionID,
		}
	}
	return cc.cache.resourcesBaseClient
}

// getSubscriptionsBaseClient returns a BaseClient instance of the subscriptions package
func (cc *CloudConnection) getSubscriptionsBaseClient() *subscriptions.BaseClient {
	if cc.cache.subscriptionsBaseClient == nil {
		cc.cache.subscriptionsBaseClient = &subscriptions.BaseClient{
			Client:  cc.client,
			BaseURI: cc.env.ResourceManagerEndpoint,
		}
	}
	return cc.cache.subscriptionsBaseClient
}

// ActivityLogsClient extends insights.ActivityLogsClient
type ActivityLogsClient struct {
	insights.ActivityLogsClient
}

// GetActivityLogsClient returns a ActivityLogsClient instance
func (cc *CloudConnection) GetActivityLogsClient() *ActivityLogsClient {
	return &ActivityLogsClient{
		insights.ActivityLogsClient{
			BaseClient: *cc.getInsightsBaseClient(),
		},
	}
}

// ContainerServicesClient extends containerservice.ContainerServicesClient
type ContainerServicesClient struct {
	containerservice.ContainerServicesClient
}

// GetContainerServicesClient returns a ContainerServicesClient instance
func (cc *CloudConnection) GetContainerServicesClient() *ContainerServicesClient {
	return &ContainerServicesClient{
		containerservice.ContainerServicesClient{
			BaseClient: *cc.getContainerServiceBaseClient(),
		},
	}
}

// GroupsClient extends resources.GroupsClient
type GroupsClient struct {
	resources.GroupsClient
}

// GetGroupsClient returns a GroupsClient instance
func (cc *CloudConnection) GetGroupsClient() *GroupsClient {
	return &GroupsClient{
		resources.GroupsClient{
			BaseClient: *cc.getResourcesBaseClient(),
		},
	}
}

// LoadBalancersClient extends network.LoadBalancersClient
type LoadBalancersClient struct {
	network.LoadBalancersClient
}

// GetLoadBalancersClient returns a LoadBalancersClient instance
func (cc *CloudConnection) GetLoadBalancersClient() *LoadBalancersClient {
	return &LoadBalancersClient{
		network.LoadBalancersClient{
			BaseClient: *cc.getNetworkBaseClient(),
		},
	}
}

// ManagedClustersClient extends containerservice.ManagedClustersClient
type ManagedClustersClient struct {
	containerservice.ManagedClustersClient
}

// GetManagedClustersClient returns a ManagedClustersClient instance
func (cc *CloudConnection) GetManagedClustersClient() *ManagedClustersClient {
	return &ManagedClustersClient{
		containerservice.ManagedClustersClient{
			BaseClient: *cc.getContainerServiceBaseClient(),
		},
	}
}

// ProvidersClient extends resources.ProvidersClient
type ProvidersClient struct {
	resources.ProvidersClient
}

// GetProvidersClient returns a ProvidersClient instance
func (cc *CloudConnection) GetProvidersClient() *ProvidersClient {
	return &ProvidersClient{
		resources.ProvidersClient{
			BaseClient: *cc.getResourcesBaseClient(),
		},
	}
}

// PublicIPAddressesClient extends network.PublicIPAddressesClient
type PublicIPAddressesClient struct {
	network.PublicIPAddressesClient
}

// GetPublicIPAddressesClient returns a PublicIPAddressesClient instance
func (cc *CloudConnection) GetPublicIPAddressesClient() *PublicIPAddressesClient {
	return &PublicIPAddressesClient{
		network.PublicIPAddressesClient{
			BaseClient: *cc.getNetworkBaseClient(),
		},
	}
}

// RoleAssignmentsClient extends authorization.RoleAssignmentsClient
type RoleAssignmentsClient struct {
	authorization.RoleAssignmentsClient
}

// GetRoleAssignmentsClient returns a RoleAssignmentsClient instance
func (cc *CloudConnection) GetRoleAssignmentsClient() *RoleAssignmentsClient {
	return &RoleAssignmentsClient{
		authorization.RoleAssignmentsClient{
			BaseClient: *cc.getAuthorizationBaseClient(),
		},
	}
}

// RoleDefinitionsClient extends authorization.RoleDefinitionsClient
type RoleDefinitionsClient struct {
	authorization.RoleDefinitionsClient
}

// GetRoleDefinitionsClient returns a RoleDefinitionsClient instance
func (cc *CloudConnection) GetRoleDefinitionsClient() *RoleDefinitionsClient {
	return &RoleDefinitionsClient{
		authorization.RoleDefinitionsClient{
			BaseClient: *cc.getAuthorizationBaseClient(),
		},
	}
}

// RouteTablesClient extends network.RouteTablesClient
type RouteTablesClient struct {
	network.RouteTablesClient
}

// GetRouteTablesClient returns a RouteTablesClient instance
func (cc *CloudConnection) GetRouteTablesClient() *RouteTablesClient {
	return &RouteTablesClient{
		network.RouteTablesClient{
			BaseClient: *cc.getNetworkBaseClient(),
		},
	}
}

// SecurityGroupsClient extends network.SecurityGroupsClient
type SecurityGroupsClient struct {
	network.SecurityGroupsClient
}

// GetSecurityGroupsClient returns a SecurityGroupsClient instance
func (cc *CloudConnection) GetSecurityGroupsClient() *SecurityGroupsClient {
	return &SecurityGroupsClient{
		network.SecurityGroupsClient{
			BaseClient: *cc.getNetworkBaseClient(),
		},
	}
}

// SubnetsClient extends network.SubnetsClient
type SubnetsClient struct {
	network.SubnetsClient
}

// GetSubnetsClient returns a SubnetsClient instance
func (cc *CloudConnection) GetSubnetsClient() *SubnetsClient {
	return &SubnetsClient{
		network.SubnetsClient{
			BaseClient: *cc.getNetworkBaseClient(),
		},
	}
}

// SubscriptionsClient extends subscriptions.Client
type SubscriptionsClient struct {
	subscriptions.Client
}

// GetSubscriptionsClient returns a SubscriptionsClient instance
func (cc *CloudConnection) GetSubscriptionsClient() *SubscriptionsClient {
	return &SubscriptionsClient{
		subscriptions.Client{
			BaseClient: *cc.getSubscriptionsBaseClient(),
		},
	}
}

// VirtualMachinesClient extends compute.VirtualMachinesClient
type VirtualMachinesClient struct {
	compute.VirtualMachinesClient
}

// GetVirtualMachinesClient returns a VirtualMachinesClient instance
func (cc *CloudConnection) GetVirtualMachinesClient() *VirtualMachinesClient {
	return &VirtualMachinesClient{
		compute.VirtualMachinesClient{
			BaseClient: *cc.getComputeBaseClient(),
		},
	}
}

// VirtualMachineScaleSetsClient extends compute.VirtualMachineScaleSetsClient
type VirtualMachineScaleSetsClient struct {
	compute.VirtualMachineScaleSetsClient
}

// GetVirtualMachineScaleSetsClient returns a VirtualMachineScaleSetsClient instance
func (cc *CloudConnection) GetVirtualMachineScaleSetsClient() *VirtualMachineScaleSetsClient {
	return &VirtualMachineScaleSetsClient{
		compute.VirtualMachineScaleSetsClient{
			BaseClient: *cc.getComputeBaseClient(),
		},
	}
}

// VirtualMachineSizesClient extends compute.VirtualMachineSizesClient
type VirtualMachineSizesClient struct {
	compute.VirtualMachineSizesClient
}

// GetVirtualMachineSizesClient returns a VirtualMachineSizesClient instance
func (cc *CloudConnection) GetVirtualMachineSizesClient() *VirtualMachineSizesClient {
	return &VirtualMachineSizesClient{
		compute.VirtualMachineSizesClient{
			BaseClient: *cc.getComputeBaseClient(),
		},
	}
}

// VirtualNetworksClient extends network.VirtualNetworksClient
type VirtualNetworksClient struct {
	network.VirtualNetworksClient
}

// GetVirtualNetworksClient returns a VirtualNetworksClient instance
func (cc *CloudConnection) GetVirtualNetworksClient() *VirtualNetworksClient {
	return &VirtualNetworksClient{
		network.VirtualNetworksClient{
			BaseClient: *cc.getNetworkBaseClient(),
		},
	}
}
