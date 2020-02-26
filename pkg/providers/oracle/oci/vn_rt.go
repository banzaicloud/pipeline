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

// CreateRouteTable creates a Route Table specified in the request
func (vn *VirtualNetwork) CreateRouteTable(request core.CreateRouteTableRequest) (table core.RouteTable, err error) {
	response, err := vn.client.CreateRouteTable(context.Background(), request)
	if err != nil {
		return table, err
	}

	return response.RouteTable, err
}

// UpdateRouteTable updates a Route Table specified in the request
func (vn *VirtualNetwork) UpdateRouteTable(request core.UpdateRouteTableRequest) (table core.RouteTable, err error) {
	response, err := vn.client.UpdateRouteTable(context.Background(), request)
	if err != nil {
		return table, err
	}

	return response.RouteTable, err
}

// DeleteRouteTable removes a Route Table by id
func (vn *VirtualNetwork) DeleteRouteTable(id *string) error {
	_, err := vn.client.DeleteRouteTable(context.Background(), core.DeleteRouteTableRequest{
		RtId: id,
	})

	return err
}

// GetRouteTable gets a Route Table by id
func (vn *VirtualNetwork) GetRouteTable(id *string) (table core.RouteTable, err error) {
	response, err := vn.client.GetRouteTable(context.Background(), core.GetRouteTableRequest{
		RtId: id,
	})

	return response.RouteTable, err
}

// GetRouteTableByName gets a Route Table by name
func (vn *VirtualNetwork) GetRouteTableByName(name string, vcnID *string) (table core.RouteTable, err error) {
	request := core.ListRouteTablesRequest{
		CompartmentId: common.String(vn.CompartmentOCID),
		DisplayName:   common.String(name),
		VcnId:         vcnID,
	}

	response, err := vn.client.ListRouteTables(context.Background(), request)
	if err != nil {
		return table, err
	}

	if len(response.Items) < 1 {
		return table, fmt.Errorf("Route Table not found: %s", name)
	}

	return response.Items[0], err
}

// GetRouteTables gets all Route Tables within a VCN
func (vn *VirtualNetwork) GetRouteTables(vcnID *string) (tables []core.RouteTable, err error) {
	request := core.ListRouteTablesRequest{
		CompartmentId: common.String(vn.CompartmentOCID),
		VcnId:         vcnID,
	}
	request.Limit = common.Int(20)

	listFunc := func(request core.ListRouteTablesRequest) (core.ListRouteTablesResponse, error) {
		return vn.client.ListRouteTables(context.Background(), request)
	}

	for response, err := listFunc(request); ; response, err = listFunc(request) {
		if err != nil {
			return tables, err
		}

		for _, item := range response.Items {
			tables = append(tables, item)
		}

		if response.OpcNextPage != nil {
			// if there are more items in next page, fetch items from next page
			request.Page = response.OpcNextPage
		} else {
			// no more result, break the loop
			break
		}
	}

	return tables, err
}
