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

package alibaba

import (
	"time"

	"emperror.dev/emperror"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/network"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
)

type alibabaNetwork struct {
	cidrs []string
	id    string
	name  string
}

func (g alibabaNetwork) CIDRs() []string {
	return g.cidrs
}

func (g alibabaNetwork) ID() string {
	return g.id
}

func (g alibabaNetwork) Name() string {
	return g.name
}

type alibabaSubnet struct {
	cidrs    []string
	id       string
	location string
	name     string
}

func (g alibabaSubnet) CIDRs() []string {
	return g.cidrs
}

func (g alibabaSubnet) ID() string {
	return g.id
}

func (g alibabaSubnet) Location() string {
	return g.location
}

func (g alibabaSubnet) Name() string {
	return g.name
}

type alibabaRouteTable struct {
	id   string
	name string
}

func (g alibabaRouteTable) ID() string {
	return g.id
}

func (g alibabaRouteTable) Name() string {
	return g.name
}

type alibabaNetworkService struct {
	logger logrus.FieldLogger
	client *vpc.Client
}

// NewNetworkService returns a new Alibaba Cloud network Service
func NewNetworkService(region string, secret *secret.SecretItemResponse, logger logrus.FieldLogger) (network.Service, error) {
	cfg := sdk.NewConfig().
		WithAutoRetry(true).
		WithDebug(true).
		WithTimeout(time.Minute)
	auth := verify.CreateAlibabaCredentials(secret.Values)
	cred := credentials.NewAccessKeyCredential(auth.AccessKeyId, auth.AccessKeySecret)
	client, err := vpc.NewClientWithOptions(region, cfg, cred)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create VPC client")
	}
	return &alibabaNetworkService{
		logger: logger,
		client: client,
	}, nil
}

// ListNetworks returns VPC networks
func (ns *alibabaNetworkService) ListNetworks() ([]network.Network, error) {
	res, err := ns.client.DescribeVpcs(vpc.CreateDescribeVpcsRequest())
	if err != nil {
		return nil, emperror.Wrap(err, "request to DescribeVpcs failed")
	}
	networks := make([]network.Network, len(res.Vpcs.Vpc))
	for idx, item := range res.Vpcs.Vpc {
		networks[idx] = &alibabaNetwork{
			cidrs: []string{item.CidrBlock},
			id:    item.VpcId,
			name:  item.VpcName,
		}
	}
	return networks, nil
}

// ListSubnets returns VPC subnetworks
func (ns *alibabaNetworkService) ListSubnets(networkID string) ([]network.Subnet, error) {
	req := vpc.CreateDescribeVSwitchesRequest()
	req.VpcId = networkID
	res, err := ns.client.DescribeVSwitches(req)
	if err != nil {
		return nil, emperror.Wrap(err, "request to DescribeVSwitches failed")
	}
	subnets := make([]network.Subnet, len(res.VSwitches.VSwitch))
	for idx, item := range res.VSwitches.VSwitch {
		subnets[idx] = &alibabaSubnet{
			cidrs:    []string{item.CidrBlock},
			id:       item.VSwitchId,
			location: item.ZoneId,
			name:     item.VSwitchName,
		}
	}
	return subnets, nil
}

// ListRouteTables returns the VPC route tables
func (ns *alibabaNetworkService) ListRouteTables(networkID string) ([]network.RouteTable, error) {
	req := vpc.CreateDescribeRouteTableListRequest()
	req.VpcId = networkID
	res, err := ns.client.DescribeRouteTableList(req)
	if err != nil {
		return nil, emperror.Wrap(err, "request to DescribeRouteTableList failed")
	}
	routeTables := make([]network.RouteTable, len(res.RouterTableList.RouterTableListType))
	for idx, item := range res.RouterTableList.RouterTableListType {
		routeTables[idx] = &alibabaRouteTable{
			id:   item.RouteTableId,
			name: item.RouteTableName,
		}
	}
	return routeTables, nil
}
