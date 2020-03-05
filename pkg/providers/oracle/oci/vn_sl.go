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

// CreateSecurityList creates a Security List specified in the request
func (vn *VirtualNetwork) CreateSecurityList(request core.CreateSecurityListRequest) (list core.SecurityList, err error) {
	response, err := vn.client.CreateSecurityList(context.Background(), request)
	if err != nil {
		return list, err
	}

	return response.SecurityList, err
}

// UpdateSecurityList updates a Security List specified in the request
func (vn *VirtualNetwork) UpdateSecurityList(request core.UpdateSecurityListRequest) (list core.SecurityList, err error) {
	response, err := vn.client.UpdateSecurityList(context.Background(), request)
	if err != nil {
		return list, err
	}

	return response.SecurityList, err
}

// DeleteSecurityList removes a Security List by id
func (vn *VirtualNetwork) DeleteSecurityList(id *string) error {
	_, err := vn.client.DeleteSecurityList(context.Background(), core.DeleteSecurityListRequest{
		SecurityListId: id,
	})

	return err
}

// GetSecurityList gets a Security List by id
func (vn *VirtualNetwork) GetSecurityList(id *string) (list core.SecurityList, err error) {
	response, err := vn.client.GetSecurityList(context.Background(), core.GetSecurityListRequest{
		SecurityListId: id,
	})

	return response.SecurityList, err
}

// GetSecurityListByName gets a Security List by name
func (vn *VirtualNetwork) GetSecurityListByName(name string, vcnID *string) (list core.SecurityList, err error) {
	request := core.ListSecurityListsRequest{
		CompartmentId: common.String(vn.CompartmentOCID),
		DisplayName:   common.String(name),
		VcnId:         vcnID,
	}

	response, err := vn.client.ListSecurityLists(context.Background(), request)
	if err != nil {
		return list, err
	}

	if len(response.Items) < 1 {
		return list, fmt.Errorf("Security List not found: %s", name)
	}

	return response.Items[0], err
}

// GetSecurityLists gets all Security Lists within a VCN
func (vn *VirtualNetwork) GetSecurityLists(vcnID *string) (lists []core.SecurityList, err error) {
	request := core.ListSecurityListsRequest{
		CompartmentId: common.String(vn.CompartmentOCID),
		VcnId:         vcnID,
	}
	request.Limit = common.Int(20)

	listFunc := func(request core.ListSecurityListsRequest) (core.ListSecurityListsResponse, error) {
		return vn.client.ListSecurityLists(context.Background(), request)
	}

	for response, err := listFunc(request); ; response, err = listFunc(request) {
		if err != nil {
			return lists, err
		}

		for _, item := range response.Items {
			lists = append(lists, item)
		}

		if response.OpcNextPage != nil {
			// if there are more items in next page, fetch items from next page
			request.Page = response.OpcNextPage
		} else {
			// no more result, break the loop
			break
		}
	}

	return lists, err
}
