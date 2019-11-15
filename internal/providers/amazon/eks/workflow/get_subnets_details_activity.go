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

package workflow

import (
	"context"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"go.uber.org/cadence/activity"

	zapadapter "logur.dev/adapter/zap"

	pkgEC2 "github.com/banzaicloud/pipeline/pkg/providers/amazon/ec2"
)

const GetSubnetsDetailsActivityName = "eks-get-subnets-details"

// GetSubnetsDetailsActivity retrieves cidr and az for subnets given their ID
type GetSubnetsDetailsActivity struct {
	awsSessionFactory *AWSSessionFactory
}

// GetSubnetsDetailsActivityInput holds IDs
// that identifies subnets which to retrieve cidr and availability zone for
type GetSubnetsDetailsActivityInput struct {
	OrganizationID uint
	SecretID       string
	Region         string

	SubnetIDs []string
}

type GetSubnetsDetailsActivityOutput struct {
	Subnets []Subnet
}

// NewGetSubnetsDetailsActivity instantiates a new NewGetSubnetsDetailsActivity
func NewGetSubnetsDetailsActivity(awsSessionFactory *AWSSessionFactory) *GetSubnetsDetailsActivity {
	return &GetSubnetsDetailsActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *GetSubnetsDetailsActivity) Execute(ctx context.Context, input GetSubnetsDetailsActivityInput) (*GetSubnetsDetailsActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"region", input.Region,
		"subnetIDs", input.SubnetIDs,
	)

	var output GetSubnetsDetailsActivityOutput
	if len(input.SubnetIDs) == 0 {
		return &output, nil
	}

	session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	netSvc := pkgEC2.NewNetworkSvc(ec2.New(session), zapadapter.New(logger.Desugar()))

	ec2Subnets, err := netSvc.GetSubnetsById(input.SubnetIDs)
	if err != nil {
		return nil, errors.WrapIf(err, "couldn't get subnets details")
	}

	for _, ec2Subnet := range ec2Subnets {
		output.Subnets = append(output.Subnets, Subnet{
			SubnetID:         aws.StringValue(ec2Subnet.SubnetId),
			Cidr:             aws.StringValue(ec2Subnet.CidrBlock),
			AvailabilityZone: aws.StringValue(ec2Subnet.AvailabilityZone),
		})
	}

	return &output, nil
}
