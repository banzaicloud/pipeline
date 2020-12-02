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
	"fmt"
	"path"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
	sdkcadence "github.com/banzaicloud/pipeline/pkg/sdk/cadence"
	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
)

const CreateAsgActivityName = "eks-create-asg"

// CreateAsgActivity responsible for creating IAM roles
type CreateAsgActivity struct {
	awsSessionFactory awsworkflow.AWSFactory
	// body of the cloud formation template for setting up the VPC
	cloudFormationTemplate string

	nodePoolStore eks.NodePoolStore
}

// CreateAsgActivityInput holds data needed for setting up IAM roles
type CreateAsgActivityInput struct {
	EKSActivityInput

	ClusterID uint

	// name of the cloud formation template stack
	StackName string

	ScaleEnabled bool
	SSHKeyName   string

	Name             string
	NodeSpotPrice    string
	Autoscaling      bool
	NodeMinCount     int
	NodeMaxCount     int
	Count            int
	NodeVolumeSize   int
	NodeImage        string
	NodeInstanceType string
	Labels           map[string]string

	Subnets             []Subnet
	VpcID               string
	SecurityGroupID     string
	NodeSecurityGroupID string

	// SecurityGroups collects the user specified custom node security group
	// IDs.
	SecurityGroups []string

	NodeInstanceRoleID string
	Tags               map[string]string
}

// CreateAsgActivityOutput holds the output data of the CreateAsgActivityOutput
type CreateAsgActivityOutput struct {
}

// CreateAsgActivity instantiates a new CreateAsgActivity
func NewCreateAsgActivity(
	awsSessionFactory awsworkflow.AWSFactory,
	cloudFormationTemplate string,
	nodePoolStore eks.NodePoolStore,
) *CreateAsgActivity {
	return &CreateAsgActivity{
		awsSessionFactory:      awsSessionFactory,
		cloudFormationTemplate: cloudFormationTemplate,
		nodePoolStore:          nodePoolStore,
	}
}

func (a *CreateAsgActivity) Execute(ctx context.Context, input CreateAsgActivityInput) (*CreateAsgActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"vpcID", input.VpcID,
		"secret", input.SecretID,
		"ami", input.NodeImage,
		"nodePool", input.Name,
	)

	awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	cloudformationClient := cloudformation.New(awsSession)

	logger.With("stackName", input.StackName).Info("creating stack")

	spotPriceParam := ""
	if p, err := strconv.ParseFloat(input.NodeSpotPrice, 64); err == nil && p > 0.0 {
		spotPriceParam = input.NodeSpotPrice
	}

	clusterAutoscalerEnabled := false
	terminationDetachEnabled := false

	if input.Autoscaling {
		clusterAutoscalerEnabled = true
	}

	// if ScaleOptions is enabled on cluster, ClusterAutoscaler is disabled on all node pools
	if input.ScaleEnabled {
		clusterAutoscalerEnabled = false
		terminationDetachEnabled = true
	}

	tags := getNodePoolStackTags(input.ClusterName, input.Tags)
	var stackParams []*cloudformation.Parameter

	// do not update node labels via kubelet boostrap params as that induces node reboot or replacement
	// we only add node pool name here, all other labels will be added by NodePoolLabelSet operator
	nodeLabels := []string{
		fmt.Sprintf("%v=%v", cluster.NodePoolNameLabelKey, input.Name),
		fmt.Sprintf("%v=%v", cluster.NodePoolVersionLabelKey, eks.CalculateNodePoolVersion(
			input.NodeImage,
			fmt.Sprintf("%d", input.NodeVolumeSize),
		)),
	}

	var subnetIDs []string

	for _, subnet := range input.Subnets {
		subnetIDs = append(subnetIDs, subnet.SubnetID)
	}

	logger.With("subnets", subnetIDs).Info("node pool mapped to subnets")

	stackParams = []*cloudformation.Parameter{
		{
			ParameterKey:   aws.String("KeyName"),
			ParameterValue: aws.String(input.SSHKeyName),
		},
		{
			ParameterKey:   aws.String("NodeImageId"),
			ParameterValue: aws.String(input.NodeImage),
		},
		{
			ParameterKey:   aws.String("NodeInstanceType"),
			ParameterValue: aws.String(input.NodeInstanceType),
		},
		{
			ParameterKey:   aws.String("NodeSpotPrice"),
			ParameterValue: aws.String(spotPriceParam),
		},
		{
			ParameterKey:   aws.String("NodeAutoScalingGroupMinSize"),
			ParameterValue: aws.String(fmt.Sprintf("%d", input.NodeMinCount)),
		},
		{
			ParameterKey:   aws.String("NodeAutoScalingGroupMaxSize"),
			ParameterValue: aws.String(fmt.Sprintf("%d", input.NodeMaxCount)),
		},
		{
			ParameterKey:   aws.String("NodeAutoScalingInitSize"),
			ParameterValue: aws.String(fmt.Sprintf("%d", input.Count)),
		},
		{
			ParameterKey:   aws.String("NodeVolumeSize"),
			ParameterValue: aws.String(fmt.Sprintf("%d", input.NodeVolumeSize)),
		},
		{
			ParameterKey:   aws.String("ClusterName"),
			ParameterValue: aws.String(input.ClusterName),
		},
		{
			ParameterKey:   aws.String("NodeGroupName"),
			ParameterValue: aws.String(input.Name),
		},
		{
			ParameterKey:   aws.String("ClusterControlPlaneSecurityGroup"),
			ParameterValue: aws.String(input.SecurityGroupID),
		},
		{
			ParameterKey:   aws.String("NodeSecurityGroup"),
			ParameterValue: aws.String(input.NodeSecurityGroupID),
		},
		{
			ParameterKey:   aws.String("CustomNodeSecurityGroups"),
			ParameterValue: aws.String(strings.Join(input.SecurityGroups, ",")),
		},
		{
			ParameterKey:   aws.String("VpcId"),
			ParameterValue: aws.String(input.VpcID),
		},
		{
			ParameterKey:   aws.String("Subnets"),
			ParameterValue: aws.String(strings.Join(subnetIDs, ",")),
		},
		{
			ParameterKey:   aws.String("NodeInstanceRoleId"),
			ParameterValue: aws.String(input.NodeInstanceRoleID),
		},
		{
			ParameterKey:   aws.String("ClusterAutoscalerEnabled"),
			ParameterValue: aws.String(fmt.Sprint(clusterAutoscalerEnabled)),
		},
		{
			ParameterKey:   aws.String("TerminationDetachEnabled"),
			ParameterValue: aws.String(fmt.Sprint(terminationDetachEnabled)),
		},
		{
			ParameterKey:   aws.String("BootstrapArguments"),
			ParameterValue: aws.String(fmt.Sprintf("--kubelet-extra-args '--node-labels %v'", strings.Join(nodeLabels, ","))),
		},
	}
	clientRequestToken := sdkAmazon.NewNormalizedClientRequestToken(input.AWSClientRequestTokenBase, CreateAsgActivityName)

	createStackInput := &cloudformation.CreateStackInput{
		ClientRequestToken: aws.String(clientRequestToken),
		DisableRollback:    aws.Bool(true),
		StackName:          aws.String(input.StackName),
		Capabilities:       []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		Parameters:         stackParams,
		Tags:               tags,
		TemplateBody:       aws.String(a.cloudFormationTemplate),
		TimeoutInMinutes:   aws.Int64(10),
	}
	createStackOutput, err := cloudformationClient.CreateStack(createStackInput)
	if err != nil {
		return nil, errors.WrapIff(err, "could not create '%s' CF stack", input.StackName)
	} else if createStackOutput == nil {
		return nil, errors.WrapIff(err, "missing stack ID for '%s' CF stack", input.StackName)
	}

	stackID := aws.StringValue(createStackOutput.StackId)
	err = a.nodePoolStore.UpdateNodePoolStackID(
		ctx,
		input.EKSActivityInput.OrganizationID,
		input.ClusterID,
		input.EKSActivityInput.ClusterName,
		input.Name,
		stackID,
	)
	if err != nil {
		return nil, errors.WrapIf(err, "updating stack ID failed")
	}

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(input.StackName)}
	err = WaitUntilStackCreateCompleteWithContext(cloudformationClient, ctx, describeStacksInput)
	if err != nil {
		return nil, packageCFError(err, input.StackName, clientRequestToken, cloudformationClient, "waiting for CF stack create operation to complete failed")
	}

	// wait for ASG fulfillment
	err = WaitForASGToBeFulfilled(ctx, logger, awsSession, input.StackName, input.Name)
	if err != nil {
		return nil, errors.WrapIff(err, "node pool %q ASG not fulfilled", input.Name)
	}

	outParams := CreateAsgActivityOutput{}
	return &outParams, nil
}

// Register registers the stored node pool deletion activity.
func (a CreateAsgActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: CreateAsgActivityName})
}

// createAsg creates an EKS autoscaling group for a node pool from the specified
// values.
//
// This is a convenience wrapper around the corresponding activity.
func createASG(
	ctx workflow.Context,
	eksActivityInput EKSActivityInput,
	eksCluster eksmodel.EKSClusterModel,
	vpcConfig GetVpcConfigActivityOutput,
	nodePool eks.NewNodePool,
	nodePoolSubnetIDs []string,
	selectedVolumeSize int,
) error {
	return createASGAsync(
		ctx, eksActivityInput,
		eksCluster, vpcConfig,
		nodePool,
		nodePoolSubnetIDs,
		selectedVolumeSize,
	).Get(ctx, nil)
}

// createAsgAsync returns a future object for creating an EKS autoscaling group
// for a node pool from the specified values.
//
// This is a convenience wrapper around the corresponding activity.
func createASGAsync(
	ctx workflow.Context,
	eksActivityInput EKSActivityInput,
	eksCluster eksmodel.EKSClusterModel,
	vpcConfig GetVpcConfigActivityOutput,
	nodePool eks.NewNodePool,
	nodePoolSubnetIDs []string,
	selectedVolumeSize int,
) workflow.Future {
	minSize := nodePool.Size
	maxSize := nodePool.Size + 1
	if nodePool.Autoscaling.Enabled {
		minSize = nodePool.Autoscaling.MinSize
		maxSize = nodePool.Autoscaling.MaxSize
	}

	sshKeyName := ""
	if eksCluster.SSHGenerated {
		sshKeyName = GenerateSSHKeyNameForCluster(eksCluster.Cluster.Name)
	}

	if nodePool.SubnetID != "" {
		if subnetIDIndex := indexStrings(nodePoolSubnetIDs, nodePool.SubnetID); subnetIDIndex == -1 {
			nodePoolSubnetIDs = append(nodePoolSubnetIDs, nodePool.SubnetID)
		}
	}

	subnets, err := NewSubnetsFromEKSSubnets(eksCluster.Subnets, nodePoolSubnetIDs...)
	if err != nil {
		return sdkcadence.NewReadyFuture(ctx, nil, errors.Wrap(err, "node pool subnets could not be determined"))
	}

	activityInput := CreateAsgActivityInput{
		EKSActivityInput: eksActivityInput,
		ClusterID:        eksCluster.Cluster.ID,

		StackName: GenerateNodePoolStackName(eksCluster.Cluster.Name, nodePool.Name),

		ScaleEnabled: eksCluster.Cluster.ScaleOptions.Enabled,
		SSHKeyName:   sshKeyName,

		Name:             nodePool.Name,
		NodeSpotPrice:    nodePool.SpotPrice,
		Autoscaling:      nodePool.Autoscaling.Enabled,
		NodeMinCount:     minSize,
		NodeMaxCount:     maxSize,
		Count:            nodePool.Size,
		NodeVolumeSize:   selectedVolumeSize,
		NodeImage:        nodePool.Image,
		NodeInstanceType: nodePool.InstanceType,
		Labels:           nodePool.Labels,

		Subnets:             subnets,
		VpcID:               vpcConfig.VpcID,
		SecurityGroupID:     vpcConfig.SecurityGroupID,
		NodeSecurityGroupID: vpcConfig.NodeSecurityGroupID,
		SecurityGroups:      nodePool.SecurityGroups,
		NodeInstanceRoleID:  path.Base(eksCluster.NodeInstanceRoleId),
		Tags:                eksCluster.Cluster.Tags,
	}

	return workflow.ExecuteActivity(ctx, CreateAsgActivityName, activityInput)
}
