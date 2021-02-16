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
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/iam"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
)

const CreateIamRolesActivityName = "eks-create-iam-roles"

// CreateIamRolesActivity responsible for creating IAM roles
type CreateIamRolesActivity struct {
	awsSessionFactory *awsworkflow.AWSSessionFactory
	// body of the cloud formation template for setting up the VPC
	cloudFormationTemplate string
}

// CreateIamRolesActivityInput holds data needed for setting up IAM roles
type CreateIamRolesActivityInput struct {
	EKSActivityInput

	// name of the cloud formation template stack
	StackName string

	DefaultUser        bool
	ClusterRoleID      string
	NodeInstanceRoleID string
	Tags               map[string]string
}

// CreateIamRolesActivityOutput holds the output data of the CreateIamRolesActivityOutput
type CreateIamRolesActivityOutput struct {
	ClusterRoleArn      string
	ClusterUserArn      string
	NodeInstanceRoleID  string
	NodeInstanceRoleArn string
}

// CreateIamRolesActivity instantiates a new CreateIamRolesActivity
func NewCreateIamRolesActivity(awsSessionFactory *awsworkflow.AWSSessionFactory, cloudFormationTemplate string) *CreateIamRolesActivity {
	return &CreateIamRolesActivity{
		awsSessionFactory:      awsSessionFactory,
		cloudFormationTemplate: cloudFormationTemplate,
	}
}

func (a *CreateIamRolesActivity) Execute(ctx context.Context, input CreateIamRolesActivityInput) (*CreateIamRolesActivityOutput, error) {
	session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	var clusterUserID string
	if input.DefaultUser {
		iamSrv := iam.New(session)
		currentUser, err := iamSrv.GetUser(&iam.GetUserInput{})
		if err != nil {
			return nil, errors.WrapIf(err, "failed to get current user (defined by secret) for IAM")
		}

		clusterUserID = aws.StringValue(currentUser.User.UserName)
	}

	userID, userPath := splitResourceId(clusterUserID)
	clusterRoleID, clusterRolePath := splitResourceId(input.ClusterRoleID)
	nodeInstanceRoleID, nodeInstanceRolePath := splitResourceId(input.NodeInstanceRoleID)

	stackParams := []*cloudformation.Parameter{
		{
			ParameterKey:   aws.String("ClusterName"),
			ParameterValue: aws.String(input.ClusterName),
		},
		{
			ParameterKey:   aws.String("UserId"),
			ParameterValue: aws.String(userID),
		},
		{
			ParameterKey:   aws.String("UserPath"),
			ParameterValue: aws.String(userPath),
		},
		{
			ParameterKey:   aws.String("ClusterRoleId"),
			ParameterValue: aws.String(clusterRoleID),
		},
		{
			ParameterKey:   aws.String("ClusterRolePath"),
			ParameterValue: aws.String(clusterRolePath),
		},
		{
			ParameterKey:   aws.String("NodeInstanceRoleId"),
			ParameterValue: aws.String(nodeInstanceRoleID),
		},
		{
			ParameterKey:   aws.String("NodeInstanceRolePath"),
			ParameterValue: aws.String(nodeInstanceRolePath),
		},
	}

	cloudformationClient := cloudformation.New(session)

	requestToken := aws.String(sdkAmazon.NewNormalizedClientRequestToken(activity.GetInfo(ctx).WorkflowExecution.ID))

	createStackInput := &cloudformation.CreateStackInput{
		ClientRequestToken: requestToken,
		DisableRollback:    aws.Bool(true),
		Capabilities: []*string{
			aws.String(cloudformation.CapabilityCapabilityIam),
			aws.String(cloudformation.CapabilityCapabilityNamedIam),
		},
		StackName:        aws.String(input.StackName),
		Parameters:       stackParams,
		Tags:             getVPCStackTags(input.ClusterName, input.Tags),
		TemplateBody:     aws.String(a.cloudFormationTemplate),
		TimeoutInMinutes: aws.Int64(10),
	}
	_, err = cloudformationClient.CreateStack(createStackInput)
	if err != nil {
		return nil, errors.WrapIf(err, "create stack failed")
	}

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(input.StackName)}
	err = WaitUntilStackCreateCompleteWithContext(cloudformationClient, ctx, describeStacksInput)
	if err != nil {
		return nil, packageCFError(err, input.StackName, *requestToken, cloudformationClient, "failed to describe stack")
	}

	describeStacksOutput, err := cloudformationClient.DescribeStacks(describeStacksInput)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get stack output parameters")
	}

	outParams := CreateIamRolesActivityOutput{}
	for _, output := range describeStacksOutput.Stacks[0].Outputs {
		switch aws.StringValue(output.OutputKey) {
		case "ClusterRoleArn":
			outParams.ClusterRoleArn = aws.StringValue(output.OutputValue)
		case "NodeInstanceRoleArn":
			outParams.NodeInstanceRoleArn = aws.StringValue(output.OutputValue)
		case "NodeInstanceRoleId":
			outParams.NodeInstanceRoleID = aws.StringValue(output.OutputValue)
		case "ClusterUserArn":
			outParams.ClusterUserArn = aws.StringValue(output.OutputValue)
		}
	}

	return &outParams, nil
}

func splitResourceId(input string) (string, string) {
	if input == "" {
		return "", ""
	}

	idSplit := strings.Split(input, "/")

	id := idSplit[len(idSplit)-1]

	path := strings.Join(idSplit[:len(idSplit)-1], "/")
	path = path + "/"
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return id, path
}
