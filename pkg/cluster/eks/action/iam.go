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

package action

import (
	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
	"github.com/banzaicloud/pipeline/utils"
)

var _ utils.RevocableAction = (*CreateIAMRolesAction)(nil)

// CreateIAMRolesAction describes the properties of cluster IAM role creation
type CreateIAMRolesAction struct {
	context   *EksClusterCreateUpdateContext
	stackName string
	log       logrus.FieldLogger
}

// NewCreateIAMRolesAction creates a new CreateIAMRolesAction
func NewCreateIAMRolesAction(log logrus.FieldLogger, creationContext *EksClusterCreateUpdateContext, stackName string) *CreateIAMRolesAction {
	return &CreateIAMRolesAction{
		context:   creationContext,
		stackName: stackName,
		log:       log,
	}
}

// GetName returns the name of this CreateIAMRolesAction
func (a *CreateIAMRolesAction) GetName() string {
	return "CreateIAMRolesAction"
}

// ExecuteAction executes this CreateIAMRolesAction
func (a *CreateIAMRolesAction) ExecuteAction(input interface{}) (interface{}, error) {
	a.log.Infoln("EXECUTE CreateIAMRolesAction, stack name:", a.stackName)

	a.log.Infoln("getting CloudFormation template for creating IAM for EKS cluster")
	templateBody, err := eks.GetIAMTemplate()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get CloudFormation template for IAM")
	}

	stackParams := []*cloudformation.Parameter{
		{
			ParameterKey:   aws.String("ClusterName"),
			ParameterValue: aws.String(a.context.ClusterName),
		},
		{
			ParameterKey:   aws.String("UserId"),
			ParameterValue: aws.String(a.context.ClusterUserID),
		},
		{
			ParameterKey:   aws.String("ClusterRoleId"),
			ParameterValue: aws.String(a.context.ClusterRoleID),
		},
	}

	if a.context.NodeInstanceRoleID != nil {
		stackParams = append(stackParams,
			&cloudformation.Parameter{
				ParameterKey:   aws.String("NodeInstanceRoleId"),
				ParameterValue: a.context.NodeInstanceRoleID,
			})
	}

	cloudformationSrv := cloudformation.New(a.context.Session)

	createStackInput := &cloudformation.CreateStackInput{
		ClientRequestToken: aws.String(uuid.Must(uuid.NewV4()).String()),
		DisableRollback:    aws.Bool(true),
		Capabilities: []*string{
			aws.String(cloudformation.CapabilityCapabilityIam),
			aws.String(cloudformation.CapabilityCapabilityNamedIam),
		},
		StackName:        aws.String(a.stackName),
		Parameters:       stackParams,
		Tags:             getVPCStackTags(a.context.ClusterName),
		TemplateBody:     aws.String(templateBody),
		TimeoutInMinutes: aws.Int64(10),
	}
	_, err = cloudformationSrv.CreateStack(createStackInput)
	if err != nil {
		return nil, errors.WrapIf(err, "create stack failed")
	}

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(a.stackName)}
	err = cloudformationSrv.WaitUntilStackCreateComplete(describeStacksInput)
	if err != nil {
		return nil, pkgCloudformation.NewAwsStackFailure(err, a.stackName, cloudformationSrv)
	}

	describeStacksOutput, err := cloudformationSrv.DescribeStacks(describeStacksInput)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get stack details", "stack", a.stackName)
	}

	var clusterRoleArn, nodeInstanceRoleArn, nodeInstanceRoleId, clusterUserArn string

	for _, output := range describeStacksOutput.Stacks[0].Outputs {
		switch aws.StringValue(output.OutputKey) {
		case "ClusterRoleArn":
			clusterRoleArn = aws.StringValue(output.OutputValue)
		case "NodeInstanceRoleArn":
			nodeInstanceRoleArn = aws.StringValue(output.OutputValue)
		case "NodeInstanceRoleId":
			nodeInstanceRoleId = aws.StringValue(output.OutputValue)
		case "ClusterUserArn":
			clusterUserArn = aws.StringValue(output.OutputValue)
		}
	}

	a.log.Infoln("cluster role ARN:", clusterRoleArn)
	a.context.ClusterRoleArn = clusterRoleArn

	a.log.Infoln("nodeInstanceRole ID:", nodeInstanceRoleId)
	a.context.NodeInstanceRoleID = &nodeInstanceRoleId

	a.log.Infoln("nodeInstanceRole ARN:", nodeInstanceRoleArn)
	a.context.NodeInstanceRoleArn = nodeInstanceRoleArn

	a.log.Infoln("cluster user ARN:", clusterUserArn)
	a.context.ClusterUserArn = clusterUserArn

	return nil, nil
}

// UndoAction rolls back this CreateIAMRolesAction
func (a *CreateIAMRolesAction) UndoAction() (err error) {
	a.log.Infoln("EXECUTE UNDO CreateIAMRolesAction, deleting stack:", a.stackName)
	cloudformationSrv := cloudformation.New(a.context.Session)
	deleteStackInput := &cloudformation.DeleteStackInput{
		ClientRequestToken: aws.String(uuid.Must(uuid.NewV4()).String()),
		StackName:          aws.String(a.stackName),
	}
	_, err = cloudformationSrv.DeleteStack(deleteStackInput)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == cloudformation.ErrCodeStackInstanceNotFoundException {
				return nil
			}
		}
	}

	return err
}
