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

package oracle

import (
	"emperror.dev/emperror"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/network"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
	secretOracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/secret"
	"github.com/banzaicloud/pipeline/secret"
)

type oracleNetwork struct {
	cidrs []string
	id    string
	name  string
}

func (g oracleNetwork) CIDRs() []string {
	return g.cidrs
}

func (g oracleNetwork) ID() string {
	return g.id
}

func (g oracleNetwork) Name() string {
	return g.name
}

type oracleSubnet struct {
	cidrs    []string
	id       string
	location string
	name     string
}

func (g oracleSubnet) CIDRs() []string {
	return g.cidrs
}

func (g oracleSubnet) ID() string {
	return g.id
}

func (g oracleSubnet) Location() string {
	return g.location
}

func (g oracleSubnet) Name() string {
	return g.name
}

type oracleRouteTable struct {
	id   string
	name string
}

func (g oracleRouteTable) ID() string {
	return g.id
}

func (g oracleRouteTable) Name() string {
	return g.name
}

type oracleNetworkService struct {
	client *oci.VirtualNetwork
	logger logrus.FieldLogger
}

// NewNetworkService returns a new Oracle network Service
func NewNetworkService(region string, secret *secret.SecretItemResponse, logger logrus.FieldLogger) (network.Service, error) {
	o, err := oci.NewOCI(secretOracle.CreateOCICredential(secret.Values))
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create OCI credential")
	}
	err = o.ChangeRegion(region)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to change OCI region")
	}
	client, err := o.NewVirtualNetworkClient()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create virtual network client")
	}
	return &oracleNetworkService{
		client: client,
		logger: logger,
	}, nil
}

// ListNetworks returns VCNs
func (ns *oracleNetworkService) ListNetworks() ([]network.Network, error) {
	vcns, err := ns.client.GetVCNs()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve VCNs")
	}
	networks := make([]network.Network, len(vcns))
	for idx, item := range vcns {
		networks[idx] = &oracleNetwork{
			cidrs: []string{deref(item.CidrBlock)},
			id:    deref(item.Id),
			name:  deref(item.DisplayName),
		}
	}
	return networks, nil
}

// ListSubnets returns VCN subnetworks
func (ns *oracleNetworkService) ListSubnets(networkID string) ([]network.Subnet, error) {
	sns, err := ns.client.GetSubnets(&networkID)
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to retrieve subnets", "networkID", networkID)
	}
	subnets := make([]network.Subnet, len(sns))
	for idx, item := range sns {
		subnets[idx] = &oracleSubnet{
			cidrs:    []string{deref(item.CidrBlock)},
			id:       deref(item.Id),
			location: deref(item.AvailabilityDomain),
			name:     deref(item.DisplayName),
		}
	}
	return subnets, nil
}

// ListRouteTables returns VCN route tables
func (ns *oracleNetworkService) ListRouteTables(networkID string) ([]network.RouteTable, error) {
	rts, err := ns.client.GetRouteTables(&networkID)
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to retrieve route tables", "networkID", networkID)
	}
	routeTables := make([]network.RouteTable, len(rts))
	for idx, item := range rts {
		routeTables[idx] = &oracleRouteTable{
			id:   deref(item.Id),
			name: deref(item.DisplayName),
		}
	}
	return routeTables, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
