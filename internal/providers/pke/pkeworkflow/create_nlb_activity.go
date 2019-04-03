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
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"go.uber.org/cadence/activity"
)

const CreateNLBActivityName = "pke-create-nlb-activity"

type CreateNLBActivity struct {
	awsClientFactory *AWSClientFactory
}

func NewCreateNLBActivity(awsClientFactory *AWSClientFactory) *CreateNLBActivity {
	return &CreateNLBActivity{
		awsClientFactory: awsClientFactory,
	}
}

type CreateNLBActivityInput struct {
	AWSActivityInput
	ClusterID   uint
	ClusterName string
	VPCID       string
	SubnetIds   []string
}

type CreateNLBActivityOutput struct {
	DNSName       string
	TargetGroup   string
	SecurityGroup string
}

func (a *CreateNLBActivity) Execute(ctx context.Context, input CreateNLBActivityInput) (*CreateNLBActivityOutput, error) {
	log := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)

	client, err := a.awsClientFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err != nil {
		return nil, err
	}

	cfClient := cloudformation.New(client)

	buf, err := ioutil.ReadFile("templates/pke/nlb.cf.yaml")
	if err != nil {
		return nil, emperror.Wrap(err, "loading CF template")
	}

	params := []*cloudformation.Parameter{
		{
			ParameterKey:   aws.String("ClusterName"),
			ParameterValue: &input.ClusterName,
		},
		{
			ParameterKey:   aws.String("VPCId"),
			ParameterValue: &input.VPCID,
		},
		{
			ParameterKey:   aws.String("SubnetIds"),
			ParameterValue: aws.String(strings.Join(input.SubnetIds, ",")),
		},
	}

	stackName := fmt.Sprintf("pke-%s-%s", "nlb", input.ClusterName)

	stackInput := &cloudformation.CreateStackInput{
		Capabilities: aws.StringSlice([]string{cloudformation.CapabilityCapabilityAutoExpand}),
		StackName:    &stackName,
		TemplateBody: aws.String(string(buf)),
		Parameters:   params,
	}

	_, err = cfClient.CreateStack(stackInput)
	if err, ok := err.(awserr.Error); ok {
		switch err.Code() {
		case cloudformation.ErrCodeAlreadyExistsException:
			log.Infof("stack already exists: %s", err.Message())

		default:
			return nil, err
		}
	}

	err = cfClient.WaitUntilStackCreateCompleteWithContext(ctx, &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)})
	if err != nil {
		return nil, emperror.Wrap(err, "failed to wait for stack")
	}

	output, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(stackName)})
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get stack")
	}

	if len(output.Stacks) != 1 {
		return nil, errors.New("couldn't find existing stack")
	}
	out := new(CreateNLBActivityOutput)
	for _, output := range output.Stacks[0].Outputs {
		switch *output.OutputKey {
		case "DNSName":
			out.DNSName = *output.OutputValue
		case "TargetGroup ":
			out.TargetGroup = *output.OutputValue
		case "SecurityGroup":
			out.SecurityGroup = *output.OutputValue
		}
	}

	return out, nil
}
