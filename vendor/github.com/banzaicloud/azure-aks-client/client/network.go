package client

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

// createNetworkInterface creates a new network interface. The Network Security Group is not a required parameter
func (a *aksClient) createNetworkInterface(rg, location, vnetName, subnetName, nsgName, ipName, nicName string) (*network.Interface, error) {

	a.LogInfo("Get VirtualNetworksClient")
	virtualNetworkClient, err := a.azureSdk.GetVirtualNetworksClient()
	if err != nil {
		return nil, err
	}

	a.LogInfo("Get IPClient")
	ipClient, err := a.azureSdk.GetIPClient()
	if err != nil {
		return nil, err
	}

	a.LogInfo("Get SubnetClient")
	subnetClient, err := a.azureSdk.GetSubnetClient()
	if err != nil {
		return nil, err
	}

	a.LogInfo("Get SecurityGroupsClient")
	securityGroupsClient, err := a.azureSdk.GetSecurityGroupsClient()
	if err != nil {
		return nil, err
	}

	a.LogInfof("Create virtual network [%s]", vnetName)
	_, err = virtualNetworkClient.CreateOrUpdateVirtualNetwork(rg, location, vnetName)
	if err != nil {
		return nil, err
	}

	a.LogInfof("Create virtual network subnet [%s]", subnetName)
	_, err = subnetClient.CreateOrUpdateVirtualNetworkSubnet(rg, vnetName, subnetName)
	if err != nil {
		return nil, err
	}

	a.LogInfof("Get virtual network subnet [%s - %s in %s]", vnetName, subnetName, rg)
	subnet, err := subnetClient.GetVirtualNetworkSubnet(rg, vnetName, subnetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get subnet: %v", err)
	}

	a.LogInfo("Create public ip")
	_, err = ipClient.CreatePublicIP(rg, location, ipName)
	if err != nil {
		return nil, err
	}

	a.LogInfof("Get public ip [%s in rg]", ipName, rg)
	ip, err := ipClient.GetPublicIP(rg, ipName)
	if err != nil {
		return nil, fmt.Errorf("failed to get ip address: %v", err)
	}

	nicParams := network.Interface{
		Name:     to.StringPtr(nicName),
		Location: to.StringPtr(location),
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: to.StringPtr("ipConfig1"),
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						Subnet:                    &subnet,
						PrivateIPAllocationMethod: network.Dynamic,
						PublicIPAddress:           &ip,
					},
				},
			},
		},
	}

	if nsgName != "" {
		a.LogInfof("create network security group [%s]", nsgName)
		_, err := securityGroupsClient.CreateOrUpdateSimpleNetworkSecurityGroup(rg, location, nsgName)
		if err != nil {
			return nil, err
		}

		a.LogInfof("Get network security group [%s]", nsgName)
		nsg, err := securityGroupsClient.GetNetworkSecurityGroup(rg, nsgName)
		if err != nil {
			return nil, err
		}
		nicParams.NetworkSecurityGroup = &nsg
	}

	a.LogInfo("Get InterfacesClient")
	nicClient, err := a.azureSdk.GetInterfacesClient()
	return nicClient.CreateOrUpdateNetworkInterface(rg, nicName, nicParams)
}
