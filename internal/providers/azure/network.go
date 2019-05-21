// Copyright © 2019 Banzai Cloud
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

package azure

import (
	"context"

	"github.com/goph/emperror"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/banzaicloud/pipeline/secret"

	intNetwork "github.com/banzaicloud/pipeline/internal/network"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type azureNetwork struct {
	cidrs []string
	id    string
	name  string
}

func (a azureNetwork) CIDRs() []string {
	return a.cidrs
}

func (a azureNetwork) ID() string {
	return a.id
}

func (a azureNetwork) Name() string {
	return a.name
}

type azureSubnet struct {
	cidrs    []string
	id       string
	location string
	name     string
}

func (a azureSubnet) CIDRs() []string {
	return a.cidrs
}

func (a azureSubnet) ID() string {
	return a.id
}

func (a azureSubnet) Location() string {
	return a.location
}

func (a azureSubnet) Name() string {
	return a.name
}

type azureRouteTable struct {
	id   string
	name string
}

func (a azureRouteTable) ID() string {
	return a.id
}

func (a azureRouteTable) Name() string {
	return a.name
}

type azureNetworkService struct {
	client            network.VirtualNetworksClient
	logger            logrus.FieldLogger
	resourceGroupName string
}

// NewNetworkService returns a new Azure network Service
func NewNetworkService(resourceGroupName string, sir *secret.SecretItemResponse, logger logrus.FieldLogger) (intNetwork.Service, error) {
	cc, err := pkgAzure.NewCloudConnection(&azure.PublicCloud, pkgAzure.NewCredentials(sir.Values))
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create cloud connection")
	}
	return &azureNetworkService{
		client:            cc.GetVirtualNetworksClient().VirtualNetworksClient,
		logger:            logger,
		resourceGroupName: resourceGroupName,
	}, nil
}

func (ns *azureNetworkService) ListNetworks() ([]intNetwork.Network, error) {
	rp, err := ns.client.List(context.TODO(), ns.resourceGroupName)
	if err != nil {
		return nil, emperror.Wrap(err, "request to list virtual networks failed")
	}
	var res []intNetwork.Network
	for rp.NotDone() {
		for _, vn := range rp.Values() {
			res = append(res, &azureNetwork{
				cidrs: *vn.AddressSpace.AddressPrefixes,
				id:    *vn.Name, // this is what we want as ID
				name:  *vn.Name,
			})
		}
		err = rp.NextWithContext(context.TODO())
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (ns *azureNetworkService) ListRouteTables(networkID string) ([]intNetwork.RouteTable, error) {
	return nil, errors.New("not implemented")
}

func (ns *azureNetworkService) ListSubnets(networkID string) ([]intNetwork.Subnet, error) {
	vn, err := ns.client.Get(context.TODO(), ns.resourceGroupName, networkID, "")
	if err != nil {
		return nil, emperror.Wrap(err, "request to get virtual network failed")
	}
	if vn.Subnets == nil {
		return nil, nil
	}
	res := make([]intNetwork.Subnet, 0, len(*vn.Subnets))
	for _, s := range *vn.Subnets {
		res = append(res, &azureSubnet{
			cidrs:    []string{*s.AddressPrefix},
			id:       *s.ID,
			name:     *s.Name,
			location: *vn.Location,
		})
	}
	return res, nil
}
