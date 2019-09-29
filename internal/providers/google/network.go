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

	"github.com/sirupsen/logrus"
	"google.golang.org/api/compute/v1"

	"github.com/banzaicloud/pipeline/internal/network"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
)

type googleNetwork struct {
	cidrs []string
	id    string
	name  string
}

func (g googleNetwork) CIDRs() []string {
	return g.cidrs
}

func (g googleNetwork) ID() string {
	return g.id
}

func (g googleNetwork) Name() string {
	return g.name
}

type googleSubnet struct {
	cidrs    []string
	id       string
	location string
	name     string
}

func (g googleSubnet) CIDRs() []string {
	return g.cidrs
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
	region         string
	serviceAccount *verify.ServiceAccount
}

// NewNetworkService returns a new Google network Service
func NewNetworkService(region string, secret *secret.SecretItemResponse, logger logrus.FieldLogger) (network.Service, error) {
	sa := verify.CreateServiceAccount(secret.Values)
	svc, err := newComputeServiceFromServiceAccount(sa)
	if err != nil {
		return nil, err
	}

	// check region exists
	_, err = svc.Regions.Get(sa.ProjectId, region).Do()
	if err != nil {
		return nil, err
	}

	ns := &googleNetworkService{
		computeService: svc,
		logger:         logger,
		region:         region,
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
			cidrs: []string{item.IPv4Range},
			id:    idToString(item.Id),
			name:  item.Name,
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
	subnetList, err := ns.computeService.Subnetworks.List(projectID, ns.region).Filter(fmt.Sprintf(`network = "%s"`, net.SelfLink)).Do()
	if err != nil {
		return nil, err
	}
	subnets := make([]network.Subnet, 0, len(subnetList.Items))
	for _, item := range subnetList.Items {
		subnets = append(subnets, &googleSubnet{
			cidrs:    []string{item.IpCidrRange},
			id:       idToString(item.Id),
			location: ns.region,
			name:     item.Name,
		})
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
