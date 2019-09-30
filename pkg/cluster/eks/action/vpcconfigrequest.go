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
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/utils"
)

var _ utils.RevocableAction = (*GenerateVPCConfigRequestAction)(nil)

// GenerateVPCConfigRequestAction describes how to request a VPC config
type GenerateVPCConfigRequestAction struct {
	context        *EksClusterCreateUpdateContext
	stackName      string
	organizationID uint
	log            logrus.FieldLogger
}

// NewGenerateVPCConfigRequestAction creates a new GenerateVPCConfigRequestAction
func NewGenerateVPCConfigRequestAction(log logrus.FieldLogger, creationContext *EksClusterCreateUpdateContext, stackName string, orgID uint) *GenerateVPCConfigRequestAction {
	return &GenerateVPCConfigRequestAction{
		context:        creationContext,
		stackName:      stackName,
		organizationID: orgID,
		log:            log,
	}
}

// GetName returns the name of this GenerateVPCConfigRequestAction
func (a *GenerateVPCConfigRequestAction) GetName() string {
	return "GenerateVPCConfigRequestAction"
}

// ExecuteAction executes this GenerateVPCConfigRequestAction
func (a *GenerateVPCConfigRequestAction) ExecuteAction(input interface{}) (interface{}, error) {
	a.log.Infoln("EXECUTE GenerateVPCConfigRequestAction, stack name:", a.stackName)
	cloudformationSrv := cloudformation.New(a.context.Session)

	describeStackResourcesInput := &cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(a.stackName),
	}

	stackResources, err := cloudformationSrv.DescribeStackResources(describeStackResourcesInput)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get stack resources", "stack", a.stackName)
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
	nodeInstanceProfileResource, found := stackResourceMap["NodeInstanceRole"]
	if !found {
		return nil, errors.New("unable to find NodeInstanceRole resource")
	}

	a.log.Infof("Stack resources: %v", stackResources)

	a.context.SecurityGroupID = securityGroupResource.PhysicalResourceId
	a.context.NodeInstanceRoleID = nodeInstanceProfileResource.PhysicalResourceId
	a.context.NodeSecurityGroupID = nodeSecurityGroup.PhysicalResourceId

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(a.stackName)}
	describeStacksOutput, err := cloudformationSrv.DescribeStacks(describeStacksInput)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get stack details", "stack", a.stackName)
	}

	var clusterRoleArn, nodeInstanceRoleArn, clusterUserArn, clusterUserAccessKeyId, clusterUserSecretAccessKey string
	var vpcId *string

	for _, output := range describeStacksOutput.Stacks[0].Outputs {
		switch aws.StringValue(output.OutputKey) {
		case "ClusterRoleArn":
			clusterRoleArn = aws.StringValue(output.OutputValue)
		case "NodeInstanceRoleArn":
			nodeInstanceRoleArn = aws.StringValue(output.OutputValue)
		case "ClusterUserArn":
			clusterUserArn = aws.StringValue(output.OutputValue)
		case "VpcId":
			vpcId = output.OutputValue
		}
	}

	clusterUserAccessKeyId, clusterUserSecretAccessKey, err = GetClusterUserAccessKeyIdAndSecretVault(a.organizationID, a.context.ClusterName)

	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve EKS cluster user access key")
	}

	a.log.Infoln("cluster VPC:", aws.StringValue(vpcId))
	a.context.VpcID = vpcId

	a.log.Infoln("cluster role ARN:", clusterRoleArn)
	a.context.ClusterRoleArn = clusterRoleArn

	a.log.Infoln("nodeInstanceRoleArn role ARN:", nodeInstanceRoleArn)
	a.context.NodeInstanceRoleArn = nodeInstanceRoleArn

	a.log.Infoln("cluster user ARN:", clusterUserArn)
	a.context.ClusterUserArn = clusterUserArn

	a.log.Infoln("cluster user access key id:", clusterUserAccessKeyId)
	a.context.ClusterUserAccessKeyId = clusterUserAccessKeyId
	a.context.ClusterUserSecretAccessKey = clusterUserSecretAccessKey

	var subnetIds []*string
	for i := range a.context.Subnets {
		subnetIds = append(subnetIds, &a.context.Subnets[i].SubnetID)
	}

	return &eks.VpcConfigRequest{
		SecurityGroupIds: []*string{a.context.SecurityGroupID},
		SubnetIds:        subnetIds,
	}, nil
}

// UndoAction rolls back this GenerateVPCConfigRequestAction
func (a *GenerateVPCConfigRequestAction) UndoAction() (err error) {
	a.log.Infoln("EXECUTE UNDO GenerateVPCConfigRequestAction, stack name:", a.stackName)
	return nil
}
