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

package pkeworkflow

import (
	"context"
	"io/ioutil"

	"emperror.dev/emperror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/providers/amazon"
)

const CreateVPCActivityName = "pke-create-vpc-activity"

type CreateVPCActivity struct {
	awsClientFactory *AWSClientFactory
}

func NewCreateVPCActivity(awsClientFactory *AWSClientFactory) *CreateVPCActivity {
	return &CreateVPCActivity{
		awsClientFactory: awsClientFactory,
	}
}

type CreateVPCActivityInput struct {
	AWSActivityInput
	ClusterID   uint
	ClusterName string
	VPCID       string
	SubnetID    string
}

func (a *CreateVPCActivity) Execute(ctx context.Context, input CreateVPCActivityInput) (string, error) {
	log := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)

	client, err := a.awsClientFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err != nil {
		return "", err
	}

	cfClient := cloudformation.New(client)

	buf, err := ioutil.ReadFile("templates/pke/vpc.cf.yaml")
	if err != nil {
		return "", emperror.Wrap(err, "loading CF template")
	}

	stackName := "pke-vpc-" + input.ClusterName
	stackInput := &cloudformation.CreateStackInput{
		Capabilities: aws.StringSlice([]string{cloudformation.CapabilityCapabilityAutoExpand}),
		StackName:    &stackName,
		TemplateBody: aws.String(string(buf)),
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("ClusterName"),
				ParameterValue: aws.String(input.ClusterName),
			},
			{
				ParameterKey:   aws.String("VpcId"),
				ParameterValue: aws.String(input.VPCID),
			},
			{
				ParameterKey:   aws.String("Subnets"),
				ParameterValue: aws.String(input.SubnetID),
			},
		},
		Tags: amazon.PipelineTags(),
	}

	output, err := cfClient.CreateStack(stackInput)
	if err, ok := err.(awserr.Error); ok {
		switch err.Code() {
		case cloudformation.ErrCodeAlreadyExistsException:
			log.Infof("stack already exists: %s", err.Message())
			return stackName, nil

		default:
			return "", err
		}
	}

	return *output.StackId, nil
}
