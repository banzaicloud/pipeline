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

package ec2

import (
	"emperror.dev/emperror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"logur.dev/logur"
)

// NetworkSvc describes the fields needed to interact with EC2 to perform network related operations
type NetworkSvc struct {
	ec2Api ec2iface.EC2API
	log    logur.Logger
}

// NewNetworkSvc instantiates a new NetworkSvc that uses the provided ec2 api to perform network related operations
func NewNetworkSvc(ec2Api ec2iface.EC2API, logger logur.Logger) *NetworkSvc {
	return &NetworkSvc{
		ec2Api: ec2Api,
		log:    logger,
	}
}

// VpcAvailable returns true of the VPC with the given id exists, and is in available state otherwise false
func (svc *NetworkSvc) VpcAvailable(vpcId string) (bool, error) {
	logger := logur.WithFields(svc.log, map[string]interface{}{"vpcId": vpcId})
	result, err := svc.ec2Api.DescribeVpcs(&ec2.DescribeVpcsInput{
		VpcIds: []*string{
			aws.String(vpcId),
		},
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String(ec2.VpcStateAvailable)},
			},
		},
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidVpcID.NotFound":
				logger.Info("VPC not found or it's not in available state")
				return false, nil
			}
		}
		return false, emperror.WrapWith(err, "failed to describe VPC", "vpcId", vpcId)
	}

	if len(result.Vpcs) == 1 {
		return true, nil
	}

	return false, nil
}

// RouteTableAvailable returns true if there is an 'active' route table with the given id and belongs to
// the VPC with the given id.
func (svc *NetworkSvc) RouteTableAvailable(routeTableId, vpcId string) (bool, error) {
	logger := logur.WithFields(svc.log, map[string]interface{}{"vpcId": vpcId, "routeTableId": routeTableId})

	result, err := svc.ec2Api.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		RouteTableIds: []*string{
			aws.String(routeTableId),
		},
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcId)},
			},
			{
				Name:   aws.String("route.state"),
				Values: []*string{aws.String(ec2.RouteStateActive)},
			},
		},
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidRouteTableID.NotFound":
				logger.Info("route table not found or it's not in active state")
				return false, nil
			}
		}
		return false, emperror.WrapWith(err, "failed to describe Route Table", "vpcId", vpcId, "routeTableId", routeTableId)
	}

	if len(result.RouteTables) == 1 {
		return true, nil
	}

	return false, nil
}

// SubnetAvailable returns true if the Subnet with given id exists and belongs to the VPC with the given id.
func (svc *NetworkSvc) SubnetAvailable(subnetId, vpcId string) (bool, error) {
	logger := logur.WithFields(svc.log, map[string]interface{}{"vpcId": vpcId, "subnetId": subnetId})
	result, err := svc.ec2Api.DescribeSubnets(&ec2.DescribeSubnetsInput{
		SubnetIds: []*string{
			aws.String(subnetId),
		},
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcId)},
			},
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String(ec2.SubnetStateAvailable)},
			},
		},
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidSubnetID.NotFound":
				logger.Info("subnet not found or it's not in available state")
				return false, nil
			}
		}
		return false, emperror.WrapWith(err, "failed to describe Subnet", "vpcId", vpcId, "subnetId", subnetId)
	}

	if len(result.Subnets) == 1 {
		return true, nil
	}

	return false, nil
}

// GetVpcDefaultSecurityGroup returns the Id of default security group of the VPC
func (svc *NetworkSvc) GetVpcDefaultSecurityGroup(vpcId string) (string, error) {
	logger := logur.WithFields(svc.log, map[string]interface{}{"vpcId": vpcId})

	result, err := svc.ec2Api.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcId)},
			},
			{
				Name:   aws.String("group-name"),
				Values: []*string{aws.String("default")},
			},
		},
	})

	if err != nil {
		return "", emperror.WrapWith(err, "failed to describe default security group of the VPC", "vpcId", vpcId)
	}

	if len(result.SecurityGroups) == 0 {
		logger.Info("VPC has no default security group")
		return "", nil
	}

	return aws.StringValue(result.SecurityGroups[0].GroupId), nil
}

// GetSubnetCidr returns the cidr of the subnet
func (svc *NetworkSvc) GetSubnetCidr(subnetId string) (string, error) {

	result, err := svc.ec2Api.DescribeSubnets(&ec2.DescribeSubnetsInput{
		SubnetIds: []*string{aws.String(subnetId)},
	})

	if err != nil {
		return "", emperror.WrapWith(err, "failed to describe subnet", "subnetId", subnetId)
	}

	if len(result.Subnets) > 0 {
		return aws.StringValue(result.Subnets[0].CidrBlock), nil
	}

	return "", nil
}

// GetSubnetsById returns the subnets identified by the provided in subnet ids
func (svc *NetworkSvc) GetSubnetsById(subnetIds []string) ([]*ec2.Subnet, error) {
	var filter []*string

	for _, subnetId := range subnetIds {
		filter = append(filter, aws.String(subnetId))
	}

	var subnets []*ec2.Subnet
	err := svc.ec2Api.DescribeSubnetsPages(
		&ec2.DescribeSubnetsInput{
			SubnetIds: filter,
		},
		func(describeSubnetsOutput *ec2.DescribeSubnetsOutput, lastPage bool) bool {
			subnets = append(subnets, describeSubnetsOutput.Subnets...)
			return lastPage
		})

	return subnets, err
}
