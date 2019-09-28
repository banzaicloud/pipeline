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

package amazon

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/network"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
)

type amazonNetwork struct {
	cidrs []string
	id    string
	name  string
}

func (a amazonNetwork) CIDRs() []string {
	return a.cidrs
}

func (a amazonNetwork) ID() string {
	return a.id
}

func (a amazonNetwork) Name() string {
	return a.name
}

type amazonSubnet struct {
	cidrs    []string
	id       string
	location string
	name     string
}

func (a amazonSubnet) CIDRs() []string {
	return a.cidrs
}

func (a amazonSubnet) ID() string {
	return a.id
}

func (a amazonSubnet) Location() string {
	return a.location
}

func (a amazonSubnet) Name() string {
	return a.name
}

type amazonRouteTable struct {
	id   string
	name string
}

func (a amazonRouteTable) ID() string {
	return a.id
}

func (a amazonRouteTable) Name() string {
	return a.name
}

type amazonNetworkService struct {
	client *ec2.EC2
	logger logrus.FieldLogger
	region string
}

// NewNetworkService returns a new Amazon network Service
func NewNetworkService(region string, secret *secret.SecretItemResponse, logger logrus.FieldLogger) (network.Service, error) {
	cred := verify.CreateAWSCredentials(secret.Values)
	client, err := verify.CreateEC2Client(cred, region)
	if err != nil {
		return nil, err
	}
	ns := amazonNetworkService{
		client: client,
		logger: logger,
		region: region,
	}
	return &ns, nil
}

func (ns *amazonNetworkService) ListNetworks() ([]network.Network, error) {
	res, err := ns.client.DescribeVpcs(&ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, err
	}
	networks := make([]network.Network, len(res.Vpcs))
	for idx, vpc := range res.Vpcs {
		networks[idx] = &amazonNetwork{
			cidrs: []string{*vpc.CidrBlock},
			id:    *vpc.VpcId,
			name:  getNameFromTags(vpc.Tags),
		}
	}
	return networks, nil
}

func (ns *amazonNetworkService) ListSubnets(networkID string) ([]network.Subnet, error) {
	res, err := ns.client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			makeNetworkIDFilter(networkID),
		},
	})
	if err != nil {
		return nil, err
	}
	subnets := make([]network.Subnet, len(res.Subnets))
	for idx, subnet := range res.Subnets {
		subnets[idx] = &amazonSubnet{
			cidrs:    []string{*subnet.CidrBlock},
			id:       *subnet.SubnetId,
			location: *subnet.AvailabilityZone,
			name:     getNameFromTags(subnet.Tags),
		}
	}
	return subnets, nil
}

func (ns *amazonNetworkService) ListRouteTables(networkID string) ([]network.RouteTable, error) {
	var routeTables []network.RouteTable
	var nextToken *string
	for {
		res, err := ns.client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
			Filters: []*ec2.Filter{
				makeNetworkIDFilter(networkID),
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		routeTables = concatRouteTables(routeTables, res.RouteTables)
		nextToken = res.NextToken
		if nextToken == nil {
			break
		}
	}
	return routeTables, nil
}

func getNameFromTags(tags []*ec2.Tag) string {
	for _, tag := range tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}
	return ""
}

func makeNetworkIDFilter(networkID string) *ec2.Filter {
	name := "vpc-id"
	return &ec2.Filter{
		Name: &name,
		Values: []*string{
			&networkID,
		},
	}
}

func concatRouteTables(dst []network.RouteTable, src []*ec2.RouteTable) []network.RouteTable {
	res := dst
	if cap(dst) < len(dst)+len(src) {
		res = make([]network.RouteTable, len(dst), len(dst)+len(src))
		copy(res, dst)
	}
	for _, routeTable := range src {
		res = append(res, &amazonRouteTable{
			id:   *routeTable.RouteTableId,
			name: getNameFromTags(routeTable.Tags),
		})
	}
	return res
}
