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
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
)

const UpdateClusterWorkflowName = "eks-update-cluster"

// UpdateClusterstructureWorkflowInput holds data needed to update EKS cluster worker node pools
type UpdateClusterstructureWorkflowInput struct {
	Region         string
	OrganizationID uint
	SecretID       string

	ClusterUID   string
	ClusterName  string
	ScaleEnabled bool

	Subnets          []Subnet
	ASGSubnetMapping map[string][]Subnet

	NodeInstanceRoleID string
	AsgList            []AutoscaleGroup
}

// UpdateClusterstructureWorkflow executes the Cadence workflow responsible for updating EKS worker nodes
func UpdateClusterWorkflow(ctx workflow.Context, input UpdateClusterstructureWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
		},
	}

	logger := workflow.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
	)

	commonActivityInput := EKSActivityInput{
		OrganizationID:            input.OrganizationID,
		SecretID:                  input.SecretID,
		Region:                    input.Region,
		ClusterName:               input.ClusterName,
		AWSClientRequestTokenBase: input.ClusterUID,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	var vpcActivityOutput GetVpcConfigActivityOutput
	{
		activityInput := &GetVpcConfigActivityInput{
			EKSActivityInput: commonActivityInput,
			StackName:        generateStackNameForCluster(input.ClusterName),
		}
		err := workflow.ExecuteActivity(ctx, GetVpcConfigActivityName, activityInput).Get(ctx, &vpcActivityOutput)
		if err != nil {
			return err
		}
	}

	asgFutures := make([]workflow.Future, 0)

	sshKeyName := generateSSHKeyNameForCluster(input.ClusterName)

	for _, nodePool := range input.AsgList {

		log := logger.With("nodePool", nodePool.Name)

		if nodePool.Delete {

			log.Info("node pool will be deleted")

			activityInput := DeleteStackActivityInput{
				EKSActivityInput: commonActivityInput,
				StackName:        GenerateNodePoolStackName(input.ClusterName, nodePool.Name),
			}
			f := workflow.ExecuteActivity(ctx, DeleteStackActivityName, activityInput)
			asgFutures = append(asgFutures, f)

		} else if nodePool.Create {

			log.Info("node pool will be created")

			asgSubnets := input.ASGSubnetMapping[nodePool.Name]
			for i := range asgSubnets {
				for _, sn := range input.Subnets {
					if (asgSubnets[i].SubnetID == "" && sn.Cidr == asgSubnets[i].Cidr) ||
						(asgSubnets[i].SubnetID != "" && sn.SubnetID == asgSubnets[i].SubnetID) {
						asgSubnets[i].SubnetID = sn.SubnetID
						asgSubnets[i].Cidr = sn.Cidr
						asgSubnets[i].AvailabilityZone = sn.AvailabilityZone
					}
				}
			}

			activityInput := CreateAsgActivityInput{
				EKSActivityInput: commonActivityInput,
				StackName:        GenerateNodePoolStackName(input.ClusterName, nodePool.Name),

				ScaleEnabled: input.ScaleEnabled,
				SSHKeyName:   sshKeyName,

				Subnets: asgSubnets,

				VpcID:               vpcActivityOutput.VpcID,
				SecurityGroupID:     vpcActivityOutput.SecurityGroupID,
				NodeSecurityGroupID: vpcActivityOutput.NodeSecurityGroupID,
				NodeInstanceRoleID:  input.NodeInstanceRoleID,

				Name:             nodePool.Name,
				NodeSpotPrice:    nodePool.NodeSpotPrice,
				Autoscaling:      nodePool.Autoscaling,
				NodeMinCount:     nodePool.NodeMinCount,
				NodeMaxCount:     nodePool.NodeMaxCount,
				Count:            nodePool.Count,
				NodeImage:        nodePool.NodeImage,
				NodeInstanceType: nodePool.NodeInstanceType,
				Labels:           nodePool.Labels,
			}
			f := workflow.ExecuteActivity(ctx, CreateAsgActivityName, activityInput)
			asgFutures = append(asgFutures, f)
		} else {
			// update nodePool
			log.Info("node pool will be updated")

			activityInput := UpdateAsgActivityInput{
				EKSActivityInput: commonActivityInput,
				StackName:        GenerateNodePoolStackName(input.ClusterName, nodePool.Name),
				ScaleEnabled:     input.ScaleEnabled,
				Name:             nodePool.Name,
				NodeSpotPrice:    nodePool.NodeSpotPrice,
				Autoscaling:      nodePool.Autoscaling,
				NodeMinCount:     nodePool.NodeMinCount,
				NodeMaxCount:     nodePool.NodeMaxCount,
				Count:            nodePool.Count,
				NodeImage:        nodePool.NodeImage,
				NodeInstanceType: nodePool.NodeInstanceType,
				Labels:           nodePool.Labels,
			}
			f := workflow.ExecuteActivity(ctx, UpdateAsgActivityName, activityInput)
			asgFutures = append(asgFutures, f)
		}
	}

	// wait for AutoScalingGroups to be created
	errs := make([]error, len(asgFutures))
	for i, future := range asgFutures {
		var activityOutput CreateAsgActivityOutput
		errs[i] = future.Get(ctx, &activityOutput)
	}
	if err := errors.Combine(errs...); err != nil {
		return err
	}

	return nil
}

func getAutoScalingGroup(cloudformationSrv *cloudformation.CloudFormation, autoscalingSrv *autoscaling.AutoScaling, stackName string) (*autoscaling.Group, error) {
	describeStackResourceInput := &cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(stackName),
	}
	stackResources, err := cloudformationSrv.DescribeStackResources(describeStackResourceInput)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get stack resources", "stack", stackName)
	}

	var asgId *string
	for _, res := range stackResources.StackResources {
		if aws.StringValue(res.LogicalResourceId) == "NodeGroup" {
			asgId = res.PhysicalResourceId
			break
		}
	}

	if asgId == nil {
		return nil, nil
	}

	describeAutoScalingGroupsInput := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			asgId,
		},
	}
	describeAutoScalingGroupsOutput, err := autoscalingSrv.DescribeAutoScalingGroups(&describeAutoScalingGroupsInput)
	if err != nil {
		return nil, err
	}

	return describeAutoScalingGroupsOutput.AutoScalingGroups[0], nil
}
