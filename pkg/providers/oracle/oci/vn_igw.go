package oci

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
)

// CreateInternetGateway creates an Internet Gateway specified in the request
func (vn *VirtualNetwork) CreateInternetGateway(request core.CreateInternetGatewayRequest) (igw core.InternetGateway, err error) {

	response, err := vn.client.CreateInternetGateway(context.Background(), request)
	if err != nil {
		return igw, err
	}

	return response.InternetGateway, err
}

// UpdateInternetGateway updates an Internet Gateway specified in the request
func (vn *VirtualNetwork) UpdateInternetGateway(request core.UpdateInternetGatewayRequest) (igw core.InternetGateway, err error) {

	response, err := vn.client.UpdateInternetGateway(context.Background(), request)
	if err != nil {
		return igw, err
	}

	return response.InternetGateway, err
}

// DeleteInternetGateway removes an Internet Gateway by id
func (vn *VirtualNetwork) DeleteInternetGateway(id *string) error {

	_, err := vn.client.DeleteInternetGateway(context.Background(), core.DeleteInternetGatewayRequest{
		IgId: id,
	})

	return err
}

// GetInternetGateway gets an Internet Gateway by id
func (vn *VirtualNetwork) GetInternetGateway(id *string) (igw core.InternetGateway, err error) {

	response, err := vn.client.GetInternetGateway(context.Background(), core.GetInternetGatewayRequest{
		IgId: id,
	})

	return response.InternetGateway, err
}

// GetInternetGatewayByName gets an Internet Gateway by name
func (vn *VirtualNetwork) GetInternetGatewayByName(name string, vcnID *string) (igw core.InternetGateway, err error) {

	request := core.ListInternetGatewaysRequest{
		CompartmentId: common.String(vn.CompartmentOCID),
		DisplayName:   common.String(name),
		VcnId:         vcnID,
	}

	response, err := vn.client.ListInternetGateways(context.Background(), request)
	if err != nil {
		return igw, err
	}

	if len(response.Items) < 1 {
		return igw, fmt.Errorf("Internet Gateway not found: %s", name)
	}

	return response.Items[0], err
}

// GetInternetGateways gets all Internet Gateways within a VCN
func (vn *VirtualNetwork) GetInternetGateways(vcnID *string) (igws []core.InternetGateway, err error) {

	request := core.ListInternetGatewaysRequest{
		CompartmentId: common.String(vn.CompartmentOCID),
		VcnId:         vcnID,
	}
	request.Limit = common.Int(20)

	listFunc := func(request core.ListInternetGatewaysRequest) (core.ListInternetGatewaysResponse, error) {
		return vn.client.ListInternetGateways(context.Background(), request)
	}

	for response, err := listFunc(request); ; response, err = listFunc(request) {
		if err != nil {
			return igws, err
		}

		for _, item := range response.Items {
			igws = append(igws, item)
		}

		if response.OpcNextPage != nil {
			// if there are more items in next page, fetch items from next page
			request.Page = response.OpcNextPage
		} else {
			// no more result, break the loop
			break
		}
	}

	return igws, err
}
