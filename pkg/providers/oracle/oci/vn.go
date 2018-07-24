package oci

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
)

// VirtualNetwork is for managing Virtual Network related calls of OCI
type VirtualNetwork struct {
	CompartmentOCID string

	oci    *OCI
	client *core.VirtualNetworkClient
}

// NewVirtualNetworkClient creates a new VirtualNetwork
func (oci *OCI) NewVirtualNetworkClient() (client *VirtualNetwork, err error) {

	client = &VirtualNetwork{}

	oClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(oci.config)
	if err != nil {
		return client, err
	}

	client.client = &oClient
	client.oci = oci
	client.CompartmentOCID = oci.CompartmentOCID

	return client, nil
}

// CreateVCN creates a VCN specified in the request
func (vn *VirtualNetwork) CreateVCN(request core.CreateVcnRequest) (vcn core.Vcn, err error) {

	response, err := vn.client.CreateVcn(context.Background(), request)
	if err != nil {
		return vcn, err
	}

	return response.Vcn, err
}

// UpdateVCN updates a VCN specified in the request
func (vn *VirtualNetwork) UpdateVCN(request core.UpdateVcnRequest) (vcn core.Vcn, err error) {

	response, err := vn.client.UpdateVcn(context.Background(), request)
	if err != nil {
		return vcn, err
	}

	return response.Vcn, err
}

// DeleteVCN deletes a VCN by id
func (vn *VirtualNetwork) DeleteVCN(id *string) error {

	_, err := vn.client.DeleteVcn(context.Background(), core.DeleteVcnRequest{
		VcnId: id,
	})

	return err
}

// GetVCN gets a VCN by id
func (vn *VirtualNetwork) GetVCN(id *string) (vcn core.Vcn, err error) {

	response, err := vn.client.GetVcn(context.Background(), core.GetVcnRequest{
		VcnId: id,
	})

	return response.Vcn, err
}

// GetVCNByName gets a VCN by name within the Compartment
func (vn *VirtualNetwork) GetVCNByName(name string) (vcn core.Vcn, err error) {

	request := core.ListVcnsRequest{
		CompartmentId: common.String(vn.CompartmentOCID),
		DisplayName:   common.String(name),
	}

	response, err := vn.client.ListVcns(context.Background(), request)
	if err != nil {
		return vcn, err
	}

	if len(response.Items) < 1 {
		return vcn, fmt.Errorf("VCN not found: %s", name)
	}

	return response.Items[0], err
}

// GetVCNs gets all VCNs within the Compartment
func (vn *VirtualNetwork) GetVCNs() (vcns []core.Vcn, err error) {

	request := core.ListVcnsRequest{
		CompartmentId: common.String(vn.CompartmentOCID),
	}
	request.Limit = common.Int(20)

	listFunc := func(request core.ListVcnsRequest) (core.ListVcnsResponse, error) {
		return vn.client.ListVcns(context.Background(), request)
	}

	for response, err := listFunc(request); ; response, err = listFunc(request) {
		if err != nil {
			return vcns, err
		}

		for _, item := range response.Items {
			vcns = append(vcns, item)
		}

		if response.OpcNextPage != nil {
			// if there are more items in next page, fetch items from next page
			request.Page = response.OpcNextPage
		} else {
			// no more result, break the loop
			break
		}
	}

	return vcns, err
}
