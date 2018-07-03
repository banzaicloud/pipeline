package oci

import (
	"context"

	"github.com/oracle/oci-go-sdk/core"
)

type VirtualNetwork struct {
	oci             *OCI
	client          *core.VirtualNetworkClient
	CompartmentOCID string
}

// NewVirtualNetworkClient creates a new VirtualNetwork
func (oci *OCI) NewVirtualNetworkClient() (client *VirtualNetwork, err error) {

	client = &VirtualNetwork{}

	oClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(oci.config)
	if err != nil {
		return client, err
	}

	client.client = &oClient
	client.CompartmentOCID = oci.CompartmentOCID

	return client, nil
}

// GetVCN gets a VCN by id
func (c *VirtualNetwork) GetVCN(id string) (VCN core.Vcn, err error) {

	request := core.GetVcnRequest{
		VcnId: &id,
	}

	r, err := c.client.GetVcn(context.Background(), request)

	return r.Vcn, err
}

// GetSubnet gets a subnet by id
func (c *VirtualNetwork) GetSubnet(id string) (Subnet core.Subnet, err error) {

	request := core.GetSubnetRequest{
		SubnetId: &id,
	}

	r, err := c.client.GetSubnet(context.Background(), request)

	return r.Subnet, err
}
