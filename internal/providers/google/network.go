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
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"google.golang.org/api/compute/v1"

	"github.com/banzaicloud/pipeline/internal/network"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
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

// ListNetworks returns VPC networks of the organization at Google
func ListNetworks(secret *secret.SecretItemResponse, logger logrus.FieldLogger) ([]network.Network, error) {
	projectID := getProjectIDFromSecret(secret)
	svc, err := newComputeServiceFromSecret(secret)
	if err != nil {
		return nil, err
	}
	networkList, err := svc.Networks.List(projectID).Do()
	if err != nil {
		return nil, err
	}
	networks := make([]network.Network, len(networkList.Items))
	for idx, item := range networkList.Items {
		networks[idx] = &googleNetwork{
			cidr: item.IPv4Range,
			id:   strconv.FormatUint(item.Id, 10),
			name: item.Name,
		}
	}
	return networks, nil
}

// ListSubnets returns VPC subnetworks of the organization in the specified VPC network at Google
func ListSubnets(secret *secret.SecretItemResponse, networkID string, logger logrus.FieldLogger) ([]network.Subnet, error) {
	projectID := getProjectIDFromSecret(secret)
	svc, err := newComputeServiceFromSecret(secret)
	if err != nil {
		return nil, err
	}
	net, err := svc.Networks.Get(projectID, networkID).Do()
	if err != nil {
		return nil, err
	}
	subnetList, err := svc.Subnetworks.AggregatedList(projectID).Filter(fmt.Sprintf(`network = "%s"`, net.SelfLink)).Do()
	if err != nil {
		return nil, err
	}
	var subnets []network.Subnet
	for _, list := range subnetList.Items {
		for _, item := range list.Subnetworks {
			subnets = append(subnets, &googleSubnet{
				cidr:     item.IpCidrRange,
				id:       strconv.FormatUint(item.Id, 10),
				location: item.Region,
				name:     item.Name,
			})
		}
	}
	return subnets, nil
}

// ListRouteTables returns the VPC route tables of the organization in the specified VPC network at Google
func ListRouteTables(secret *secret.SecretItemResponse, networkID string, logger logrus.FieldLogger) ([]network.RouteTable, error) {
	projectID := getProjectIDFromSecret(secret)
	svc, err := newComputeServiceFromSecret(secret)
	if err != nil {
		return nil, err
	}
	net, err := svc.Networks.Get(projectID, networkID).Do()
	if err != nil {
		return nil, err
	}
	routeList, err := svc.Routes.List(projectID).Filter(fmt.Sprintf(`network = "%s"`, net.SelfLink)).Do()
	if err != nil {
		return nil, err
	}
	routeTables := make([]network.RouteTable, len(routeList.Items))
	for idx, item := range routeList.Items {
		routeTables[idx] = &googleRouteTable{
			id:   strconv.FormatUint(item.Id, 10),
			name: item.Name,
		}
	}
	return routeTables, nil
}

func newComputeServiceFromSecret(secret *secret.SecretItemResponse) (*compute.Service, error) {
	serviceAccount := verify.CreateServiceAccount(secret.Values)
	jsonConfig, err := json.Marshal(serviceAccount)
	if err != nil {
		return nil, err
	}
	jwtConf, err := google.JWTConfigFromJSON(jsonConfig, compute.ComputeReadonlyScope)
	if err != nil {
		return nil, err
	}
	client := jwtConf.Client(context.Background())
	return compute.New(client)
}

func getProjectIDFromSecret(secret *secret.SecretItemResponse) string {
	return secret.GetValue("project_id")
}
