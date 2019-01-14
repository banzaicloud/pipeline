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

package google

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/api/compute/v1"

	"github.com/banzaicloud/pipeline/internal/network"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/sirupsen/logrus"
)

type googleNetwork struct {
	cidr string
	id   string
	name string
}

func (g googleNetwork) CIDR() string {
	return g.cidr
}

func (g googleNetwork) ID() string {
	return g.id
}

func (g googleNetwork) Name() string {
	return g.name
}

type googleSubnet struct {
	cidr     string
	id       string
	location string
	name     string
}

func (g googleSubnet) CIDR() string {
	return g.cidr
}

func (g googleSubnet) ID() string {
	return g.id
}

func (g googleSubnet) Location() string {
	return g.location
}

func (g googleSubnet) Name() string {
	return g.name
}

type googleRouteTable struct {
	id   string
	name string
}

func (g googleRouteTable) ID() string {
	return g.id
}

func (g googleRouteTable) Name() string {
	return g.name
}

type googleNetworkService struct {
	computeService *compute.Service
	logger         logrus.FieldLogger
	serviceAccount *verify.ServiceAccount
}

// NewNetworkService returns a new Google network Service
func NewNetworkService(secret *secret.SecretItemResponse, logger logrus.FieldLogger) (network.Service, error) {
	sa := verify.CreateServiceAccount(secret.Values)
	svc, err := newComputeServiceFromServiceAccount(sa)
	if err != nil {
		return nil, err
	}
	ns := &googleNetworkService{
		computeService: svc,
		logger:         logger,
		serviceAccount: sa,
	}
	return ns, nil
}

// ListNetworks returns VPC networks of the project at Google
func (ns *googleNetworkService) ListNetworks() ([]network.Network, error) {
	networkList, err := ns.computeService.Networks.List(ns.serviceAccount.ProjectId).Do()
	if err != nil {
		return nil, err
	}
	networks := make([]network.Network, len(networkList.Items))
	for idx, item := range networkList.Items {
		networks[idx] = &googleNetwork{
			cidr: item.IPv4Range,
			id:   idToString(item.Id),
			name: item.Name,
		}
	}
	return networks, nil
}

// ListSubnets returns VPC subnetworks of the organization in the specified VPC network at Google
func (ns *googleNetworkService) ListSubnets(networkID string) ([]network.Subnet, error) {
	projectID := ns.serviceAccount.ProjectId
	net, err := ns.computeService.Networks.Get(projectID, networkID).Do()
	if err != nil {
		return nil, err
	}
	subnetList, err := ns.computeService.Subnetworks.AggregatedList(projectID).Filter(fmt.Sprintf(`network = "%s"`, net.SelfLink)).Do()
	if err != nil {
		return nil, err
	}
	var subnets []network.Subnet
	for region, list := range subnetList.Items {
		location := strings.TrimPrefix(region, "regions/")
		for _, item := range list.Subnetworks {
			subnets = append(subnets, &googleSubnet{
				cidr:     item.IpCidrRange,
				id:       idToString(item.Id),
				location: location,
				name:     item.Name,
			})
		}
	}
	return subnets, nil
}

// ListRouteTables returns the VPC route tables of the organization in the specified VPC network at Google
func (ns *googleNetworkService) ListRouteTables(networkID string) ([]network.RouteTable, error) {
	projectID := ns.serviceAccount.ProjectId
	net, err := ns.computeService.Networks.Get(projectID, networkID).Do()
	if err != nil {
		return nil, err
	}
	routeList, err := ns.computeService.Routes.List(projectID).Filter(fmt.Sprintf(`network = "%s"`, net.SelfLink)).Do()
	if err != nil {
		return nil, err
	}
	routeTables := make([]network.RouteTable, len(routeList.Items))
	for idx, item := range routeList.Items {
		routeTables[idx] = &googleRouteTable{
			id:   idToString(item.Id),
			name: item.Name,
		}
	}
	return routeTables, nil
}

func newComputeServiceFromServiceAccount(serviceAccount *verify.ServiceAccount) (*compute.Service, error) {
	client, err := verify.CreateOath2Client(serviceAccount, compute.ComputeReadonlyScope)
	if err != nil {
		return nil, err
	}
	return compute.New(client)
}

func idToString(id uint64) string {
	return strconv.FormatUint(id, 10)
}
