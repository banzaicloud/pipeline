package compute

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"context"
)

// VirtualMachineSizesClient responsible for VMSize
type VirtualMachineSizesClient struct {
	client *compute.VirtualMachineSizesClient
}

// NewVirtualMachineSizesClient create a new 'VirtualMachineSizesClient' instance
func NewVirtualMachineSizesClient(authorizer autorest.Authorizer, subscriptionId string) *VirtualMachineSizesClient {
	vmSizesClient := compute.NewVirtualMachineSizesClient(subscriptionId)
	vmSizesClient.Authorizer = authorizer

	return &VirtualMachineSizesClient{
		client: &vmSizesClient,
	}
}

// ListVirtualMachineSizes lists all supported VM size in the given location
func (vms *VirtualMachineSizesClient) ListVirtualMachineSizes(location string) (*compute.VirtualMachineSizeListResult, error) {
	list, err := vms.client.List(context.Background(), location)
	if err != nil {
		return nil, err
	}
	return &list, nil
}
