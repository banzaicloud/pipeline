package client

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	clientCompute "github.com/banzaicloud/azure-aks-client/service/compute"
)

// createVirtualMachine creates a virtual machine with a systems assigned identity type
func (a *aksClient) createVirtualMachine(rg, location, vnetName, subnetName, nsgName, ipName, vmName, nicName string) (*compute.VirtualMachine, error) {

	a.LogInfo("Get InterfacesClient")
	nicClient, err := a.azureSdk.GetInterfacesClient()
	if err != nil {
		return nil, err
	}

	a.LogInfo("Get VirtualMachineClient")
	vmClient, err := a.azureSdk.GetVirtualMachineClient()
	if err != nil {
		return nil, err
	}

	a.LogInfof("Create network interface [%s %s %s %s %s %s %s]", rg, location, vnetName, subnetName, nsgName, ipName, nicName)
	_, err = a.createNetworkInterface(rg, location, vnetName, subnetName, nsgName, ipName, nicName)
	if err != nil {
		return nil, err
	}

	a.LogInfof("Get network interface [%s] in [%s]", nicName, rg)
	nic, err := nicClient.GetNetworkInterface(rg, nicName)
	if err != nil {
		return nil, err
	}

	a.LogInfo("Create virtual machine [%s] in [%s]", vmName, rg)
	return vmClient.CreateVirtualMachine(&clientCompute.CreateVmRequest{
		ResourceGroup:      rg,
		Location:           location,
		IpName:             ipName,
		VmName:             vmName,
		Publisher:          "Canonical",
		Offer:              "UbuntuServer",
		Sku:                "16.04.0-LTS",
		NetworkInterfaceId: *nic.ID,
	})
}

// getVirtualMachine retrieves information about a virtual machine
func (a *aksClient) getVirtualMachine(resourceGroup, clusterName, location, vmName string) (*compute.VirtualMachine, error) {
	a.LogInfo("Get VirtualMachineClient")
	vmClient, err := a.azureSdk.GetVirtualMachineClient()
	if err != nil {
		return nil, err
	}

	a.LogInfo("Find infrastructure resource group [%s %s %s]", resourceGroup, clusterName, location)
	infrastructureRg, err := a.findInfrastructureResourceGroup(resourceGroup, clusterName, location)
	if err != nil {
		return nil, err
	}
	a.LogInfo("Infrastructure resource group %s", *infrastructureRg.Name)

	return vmClient.GetVirtualMachine(*infrastructureRg.Name, vmName)
}

// ListVirtualMachines lists all of the virtual machines in the specified resource group
func (a *aksClient) listVirtualMachines(resourceGroup, clusterName, location string) ([]compute.VirtualMachine, error) {
	a.LogInfo("Get VirtualMachineClient")
	vmClient, err := a.azureSdk.GetVirtualMachineClient()
	if err != nil {
		return nil, err
	}

	a.LogInfof("Find infrastructure resource group[%s, %s, %s]", resourceGroup, clusterName, location)

	infrastructureRg, err := a.findInfrastructureResourceGroup(resourceGroup, clusterName, location)
	if err != nil {
		return nil, err
	}
	a.LogInfof("Infrastructure resource group: %s", *infrastructureRg.Name)

	return vmClient.ListVirtualMachines(*infrastructureRg.Name)
}

// enableManagedServiceIdentity enables the Managed Service Identity on the given virtual machine
func (a *aksClient) enableManagedServiceIdentity(resourceGroup, clusterName, location string) error {
	a.LogInfo("Get VirtualMachineClient")
	vmClient, err := a.azureSdk.GetVirtualMachineClient()
	if err != nil {
		return err
	}

	a.LogInfo("List virtual machines in [%s]", resourceGroup)
	vmList, err := a.listVirtualMachines(resourceGroup, clusterName, location)

	for _, vm := range vmList {
		a.LogInfo("Enable MSI in VM[%s]", *vm.Name)
		_, err := vmClient.EnableManagedServiceIdentity(&vm, resourceGroup, location)
		if err != nil {
			return err
		}
	}
	return nil
}

// disableManagedServiceIdentity disables the Managed Service Identity on the given virtual machine
func (a *aksClient) disableManagedServiceIdentity(resourceGroup, clusterName, location string) error {
	a.LogInfo("Get VirtualMachineClient")
	vmClient, err := a.azureSdk.GetVirtualMachineClient()
	if err != nil {
		return err
	}

	a.LogInfo("List virtual machines in [%s]", resourceGroup)
	vmList, err := a.listVirtualMachines(resourceGroup, clusterName, location)

	for _, vm := range vmList {
		a.LogInfof("Disable MSI in VM[%s]", *vm.Name)
		_, err := vmClient.DisableManagedServiceIdentity(&vm, resourceGroup, location)
		if err != nil {
			return err
		}
	}
	return nil
}

// listVirtualMachineSizes lists all supported vm size in the given location
func (a *aksClient) listVirtualMachineSizes(location string) ([]compute.VirtualMachineSize, error) {
	a.LogInfo("Get VirtualMachineClient")
	vmSizeClient, err := a.azureSdk.GetVirtualMachineSizesClient()
	if err != nil {
		return nil, err
	}

	a.LogInfof("List VM sizes in [%s] location", location)
	res, err := vmSizeClient.ListVirtualMachineSizes(location)
	if err != nil {
		return nil, err
	}

	return *res.Value, nil
}
