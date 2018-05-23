package compute

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"fmt"
	"context"
	"github.com/banzaicloud/azure-aks-client/utils"
)

// VirtualMachinesClient responsible for the VM
type VirtualMachinesClient struct {
	client *compute.VirtualMachinesClient
}

// CreateVmRequest describes a VM create request
type CreateVmRequest struct {
	ResourceGroup      string
	Location           string
	IpName             string
	VmName             string
	Publisher          string
	Offer              string
	Sku                string
	NetworkInterfaceId string
}

// NewVirtualMachinesClient create a new 'VirtualMachinesClient' instance
func NewVirtualMachinesClient(authorizer autorest.Authorizer, subscriptionId string) *VirtualMachinesClient {
	virtualMachinesClient := compute.NewVirtualMachinesClient(subscriptionId)
	virtualMachinesClient.Authorizer = authorizer

	return &VirtualMachinesClient{
		client: &virtualMachinesClient,
	}
}

// CreateVirtualMachine creates a virtual machine with a systems assigned identity type
func (vmc *VirtualMachinesClient) CreateVirtualMachine(r *CreateVmRequest) (*compute.VirtualMachine, error) {

	future, err := vmc.client.CreateOrUpdate(
		context.Background(),
		r.ResourceGroup,
		r.VmName,
		compute.VirtualMachine{
			Location: to.StringPtr(r.Location),
			Identity: &compute.VirtualMachineIdentity{
				Type: compute.ResourceIdentityTypeSystemAssigned, // needed to add MSI authentication
			},
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				HardwareProfile: &compute.HardwareProfile{
					VMSize: compute.VirtualMachineSizeTypesBasicA0,
				},
				StorageProfile: &compute.StorageProfile{
					ImageReference: &compute.ImageReference{
						Publisher: to.StringPtr(r.Publisher),
						Offer:     to.StringPtr(r.Offer),
						Sku:       to.StringPtr(r.Sku),
						Version:   to.StringPtr("latest"),
					},
				},
				OsProfile: &compute.OSProfile{
					AdminUsername: utils.S("pipeline"),
					LinuxConfiguration: &compute.LinuxConfiguration{
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{
								{
									KeyData: utils.S(utils.ReadPubRSA("id_rsa.pub")),
								},
							},
						},
					},
				},
				NetworkProfile: &compute.NetworkProfile{
					NetworkInterfaces: &[]compute.NetworkInterfaceReference{
						{
							ID: to.StringPtr(r.NetworkInterfaceId),
							NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
								Primary: to.BoolPtr(true),
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create vm: %v", err)
	}

	err = future.WaitForCompletion(context.Background(), vmc.client.Client)
	if err != nil {
		return nil, fmt.Errorf("cannot get the vm create or update future response: %v", err)
	}

	vm, err := future.Result(*vmc.client)
	if err != nil {
		return nil, err
	}

	return &vm, err
}

// DisableManagedServiceIdentity disables the Managed Service Identity on the given virtual machine
func (vmc *VirtualMachinesClient) DisableManagedServiceIdentity(existsVM *compute.VirtualMachine, rg, location string) (*compute.VirtualMachine, error) {
	return vmc.updateMSI(existsVM, rg, location, false)
}

// EnableManagedServiceIdentity enables the Managed Service Identity on the given virtual machine
func (vmc *VirtualMachinesClient) EnableManagedServiceIdentity(existsVM *compute.VirtualMachine, rg, location string) (*compute.VirtualMachine, error) {
	return vmc.updateMSI(existsVM, rg, location, true)
}

// updateMSI enables or disables the Managed Service Identity on the given virtual machine
func (vmc *VirtualMachinesClient) updateMSI(existsVM *compute.VirtualMachine, rg, location string, isMSIEnabled bool) (*compute.VirtualMachine, error) {

	var identityType compute.ResourceIdentityType
	if isMSIEnabled {
		identityType = compute.ResourceIdentityTypeSystemAssigned
	} else {
		identityType = compute.ResourceIdentityTypeNone
	}

	existsVM.Resources = nil
	existsVM.Identity = &compute.VirtualMachineIdentity{
		Type: identityType,
	}

	return vmc.CreateOrUpdateVirtualMachine(rg, *existsVM.Name, *existsVM)
}

// CreateOrUpdateVirtualMachine creates or updates a virtual machine
func (vmc *VirtualMachinesClient) CreateOrUpdateVirtualMachine(resourceGroup, vmName string, params compute.VirtualMachine) (*compute.VirtualMachine, error) {
	future, err := vmc.client.CreateOrUpdate(context.Background(), resourceGroup, vmName, params)
	if err != nil {
		return nil, err
	}

	err = future.WaitForCompletion(context.Background(), vmc.client.Client)
	if err != nil {
		return nil, fmt.Errorf("cannot get the vm create or update future response: %v", err)
	}

	vm, err := future.Result(*vmc.client)
	if err != nil {
		return nil, err
	}
	return &vm, nil
}

// ListVirtualMachines lists all of the virtual machines in the specified resource group
func (vmc *VirtualMachinesClient) ListVirtualMachines(infrastructureResourceGroup string) ([]compute.VirtualMachine, error) {
	page, err := vmc.client.List(context.Background(), infrastructureResourceGroup)
	if err != nil {
		return nil, err
	}
	return page.Values(), nil
}

// GetVirtualMachine retrieves information about a virtual machine
func (vmc *VirtualMachinesClient) GetVirtualMachine(rg, vmName string) (*compute.VirtualMachine, error) {
	vm, err := vmc.client.Get(context.Background(), rg, vmName, "")
	if err != nil {
		return nil, err
	}
	return &vm, err
}
