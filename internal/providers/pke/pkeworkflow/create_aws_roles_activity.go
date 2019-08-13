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
	"github.com/banzaicloud/pipeline/internal/providers/amazon"
	"go.uber.org/cadence/activity"
)

const CreateAWSRolesActivityName = "pke-create-aws-roles-activity"
const PkeGlobalStackName = "pke-global"

type CreateAWSRolesActivity struct {
	awsClientFactory *AWSClientFactory
}

func NewCreateAWSRolesActivity(awsClientFactory *AWSClientFactory) *CreateAWSRolesActivity {
	return &CreateAWSRolesActivity{
		awsClientFactory: awsClientFactory,
	}
}

type CreateAWSRolesActivityInput struct {
	AWSActivityInput
	ClusterID uint
}

func (a *CreateAWSRolesActivity) Execute(ctx context.Context, input CreateAWSRolesActivityInput) (string, error) {
	logger := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)

	client, err := a.awsClientFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err != nil {
		return "", err
	}

	cfClient := cloudformation.New(client)

	buf, err := ioutil.ReadFile("templates/pke/global.cf.yaml")
	if err != nil {
		return "", emperror.Wrap(err, "loading CF template")
	}

	stackInput := &cloudformation.CreateStackInput{
		Capabilities: aws.StringSlice([]string{cloudformation.CapabilityCapabilityIam, cloudformation.CapabilityCapabilityNamedIam}),
		StackName:    aws.String(PkeGlobalStackName),
		TemplateBody: aws.String(string(buf)),
		Tags:         amazon.PipelineTags(),
	}

	output, err := cfClient.CreateStack(stackInput)
	if err, ok := err.(awserr.Error); ok {
		switch err.Code() {
		case cloudformation.ErrCodeAlreadyExistsException:
			logger.Infof("stack already exists: %s", err.Message())
			return PkeGlobalStackName, nil
		default:
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	return *output.StackId, nil
}
