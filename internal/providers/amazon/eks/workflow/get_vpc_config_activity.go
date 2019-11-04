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

package workflow

import (
	"context"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

const GetVpcConfigActivityName = "eks-get-vpc-cfg"

// GetVpcConfigActivity responsible for creating IAM roles
type GetVpcConfigActivity struct {
	awsSessionFactory *AWSSessionFactory
}

// GetVpcConfigActivityInput holds data needed for setting up IAM roles
type GetVpcConfigActivityInput struct {
	EKSActivityInput

	// name of the cloud formation template stack
	StackName string
}

// GetVpcConfigActivityOutput holds the output data of the GetVpcConfigActivityOutput
type GetVpcConfigActivityOutput struct {
	SecurityGroupID     string
	NodeSecurityGroupID string
}

// GetVpcConfigActivity instantiates a new GetVpcConfigActivity
func NewGetVpcConfigActivity(awsSessionFactory *AWSSessionFactory) *GetVpcConfigActivity {
	return &GetVpcConfigActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *GetVpcConfigActivity) Execute(ctx context.Context, input GetVpcConfigActivityInput) (*GetVpcConfigActivityOutput, error) {

	session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	cloudformationClient := cloudformation.New(session)

	describeStackResourcesInput := &cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(input.StackName),
	}

	stackResources, err := cloudformationClient.DescribeStackResources(describeStackResourcesInput)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get stack resources", "stack", input.StackName)
	}

	stackResourceMap := make(map[string]cloudformation.StackResource)
	for _, res := range stackResources.StackResources {
		stackResourceMap[*res.LogicalResourceId] = *res
	}

	securityGroupResource, found := stackResourceMap["ControlPlaneSecurityGroup"]
	if !found {
		return nil, errors.New("unable to find ControlPlaneSecurityGroup resource")
	}
	nodeSecurityGroup, found := stackResourceMap["NodeSecurityGroup"]
	if !found {
		return nil, errors.New("unable to find NodeSecurityGroup resource")
	}

	output := GetVpcConfigActivityOutput{}
	output.SecurityGroupID = aws.StringValue(securityGroupResource.PhysicalResourceId)
	output.NodeSecurityGroupID = aws.StringValue(nodeSecurityGroup.PhysicalResourceId)

	return &output, nil
}
