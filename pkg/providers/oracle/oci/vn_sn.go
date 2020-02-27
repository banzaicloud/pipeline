// Copyright © 2018 Banzai Cloud
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

package oci

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
)

// CreateSubnet creates a Subnet specified in the request
func (vn *VirtualNetwork) CreateSubnet(request core.CreateSubnetRequest) (subnet core.Subnet, err error) {
	response, err := vn.client.CreateSubnet(context.Background(), request)
	if err != nil {
		return subnet, err
	}

	return response.Subnet, err
}

// UpdateSubnet updates a Subnet specified in the request
func (vn *VirtualNetwork) UpdateSubnet(request core.UpdateSubnetRequest) (subnet core.Subnet, err error) {
	response, err := vn.client.UpdateSubnet(context.Background(), request)
	if err != nil {
		return subnet, err
	}

	return response.Subnet, err
}

// DeleteSubnet removes a Subnet by id
func (vn *VirtualNetwork) DeleteSubnet(id *string) error {
	_, err := vn.client.DeleteSubnet(context.Background(), core.DeleteSubnetRequest{
		SubnetId: id,
	})

	return err
}

// GetSubnet gets a Subnet by id
func (vn *VirtualNetwork) GetSubnet(id *string) (subnet core.Subnet, err error) {
	response, err := vn.client.GetSubnet(context.Background(), core.GetSubnetRequest{
		SubnetId: id,
	})

	return response.Subnet, err
}

// GetSubnetByName gets a Subnet by name within a VCN
func (vn *VirtualNetwork) GetSubnetByName(name string, vcnID *string) (subnet core.Subnet, err error) {
	request := core.ListSubnetsRequest{
		CompartmentId: common.String(vn.CompartmentOCID),
		DisplayName:   common.String(name),
		VcnId:         vcnID,
	}

	response, err := vn.client.ListSubnets(context.Background(), request)
	if err != nil {
		return subnet, err
	}

	if len(response.Items) < 1 {
		return subnet, fmt.Errorf("Subnet not found: %s", name)
	}

	return response.Items[0], err
}

// GetSubnets gets all Subnets within a VCN
func (vn *VirtualNetwork) GetSubnets(vcnID *string) (subnets []core.Subnet, err error) {
	request := core.ListSubnetsRequest{
		CompartmentId: common.String(vn.CompartmentOCID),
		VcnId:         vcnID,
	}
	request.Limit = common.Int(20)

	listFunc := func(request core.ListSubnetsRequest) (core.ListSubnetsResponse, error) {
		return vn.client.ListSubnets(context.Background(), request)
	}

	for response, err := listFunc(request); ; response, err = listFunc(request) {
		if err != nil {
			return subnets, err
		}

		for _, item := range response.Items {
			subnets = append(subnets, item)
		}

		if response.OpcNextPage != nil {
			// if there are more items in next page, fetch items from next page
			request.Page = response.OpcNextPage
		} else {
			// no more result, break the loop
			break
		}
	}

	return subnets, err
}
