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
	"fmt"
	"math"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/workflow"
)

const UpdateClusterWorkflowName = "pke-update-cluster"

type UpdateClusterWorkflowInput struct {
	OrganizationID              uint
	ClusterID                   uint
	ClusterUID                  string
	ClusterName                 string
	SecretID                    string
	Region                      string
	PipelineExternalURL         string
	PipelineExternalURLInsecure bool
	NodePoolsToAdd              []NodePool
	NodePoolsToDelete           []NodePool
	NodePoolsToUpdate           []NodePool
	VPCID                       string
	SubnetIDs                   []string
}

func UpdateClusterWorkflow(ctx workflow.Context, input UpdateClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// Generic AWS activity input
	awsActivityInput := AWSActivityInput{
		OrganizationID: input.OrganizationID,
		SecretID:       input.SecretID,
		Region:         input.Region,
	}

	var masterOutput map[string]string
	err := workflow.ExecuteActivity(ctx,
		WaitCFCompletionActivityName,
		WaitCFCompletionActivityInput{
			AWSActivityInput: awsActivityInput,
			StackID:          "pke-master-" + input.ClusterName,
		}).Get(ctx, &masterOutput)
	if err != nil {
		return err
	}
	clusterSecurityGroup := masterOutput["ClusterSecurityGroup"]

	// Get default security group of the VPC
	var vpcDefaultSecurityGroupID string

	activityInput := GetVpcDefaultSecurityGroupActivityInput{
		AWSActivityInput: awsActivityInput,
		ClusterID:        input.ClusterID,
		VpcID:            input.VPCID,
	}
	err = workflow.ExecuteActivity(ctx, GetVpcDefaultSecurityGroupActivityName, activityInput).Get(ctx, &vpcDefaultSecurityGroupID)
	if err != nil {
		return err
	}

	// delete removed nodepools
	{
		futures := make([]workflow.Future, 0, 2*len(input.NodePoolsToDelete))
		errs := make([]error, 0, len(futures))

		for _, np := range input.NodePoolsToDelete {
			if np.Master || !np.Worker {
				continue
			}

			activityInput := DeletePoolActivityInput{
				// AWSActivityInput: awsActivityInput,
				ClusterID: input.ClusterID,
				Pool:      np,
			}

			// initiate deletion
			if e := workflow.ExecuteActivity(ctx, DeletePoolActivityName, activityInput).Get(ctx, nil); err != nil {
				errs = append(errs, errors.Wrapf(e, "couldn't initiate master node pool deletion"))
				continue
			}

			futures = append(futures, workflow.ExecuteActivity(ctx, WaitForDeletePoolActivityName, activityInput))
		}

		for _, future := range futures {
			if future != nil {
				if e := future.Get(ctx, nil); e != nil {
					errs = append(errs, errors.Wrap(e, "couldn't delete node pool"))
				}
			}
		}

		if err := errors.Combine(errs...); err != nil {
			return err
		}
	}

	asgIDs := make([]string, len(input.NodePoolsToUpdate))

	// create/change nodepools that are not removed
	{
		futures := make([]workflow.Future, len(input.NodePoolsToUpdate))

		for i, np := range input.NodePoolsToUpdate {
			stackName := fmt.Sprintf("pke-pool-%s-worker-%s", input.ClusterName, np.Name)

			activityInput := WaitCFCompletionActivityInput{
				AWSActivityInput: awsActivityInput,
				StackID:          stackName,
			}

			futures[i] = workflow.ExecuteActivity(ctx, WaitCFCompletionActivityName, activityInput)
		}

		errs := make([]error, len(futures))
		for i, future := range futures {
			var cfOut map[string]string
			err := future.Get(ctx, &cfOut)
			if err != nil {
				errs[i] = errors.Wrapf(err, "can't find AutoScalingGroup for pool %q", input.NodePoolsToUpdate[i].Name)

				continue
			}

			asgID, ok := cfOut["AutoScalingGroupId"]
			if !ok {
				errs[i] = errors.Errorf("can't find AutoScalingGroup for pool %q", input.NodePoolsToUpdate[i].Name)

				continue
			}

			asgIDs[i] = asgID
		}

		if err := errors.Combine(errs...); err != nil {
			return err
		}
	}

	{
		futures := make([]workflow.Future, len(input.NodePoolsToUpdate))

		for i, np := range input.NodePoolsToUpdate {
			activityInput := UpdatePoolActivityInput{
				AWSActivityInput: awsActivityInput,
				Pool:             np,
				AutoScalingGroup: asgIDs[i],
			}

			futures[i] = workflow.ExecuteActivity(ctx, UpdatePoolActivityName, activityInput)
		}

		errs := make([]error, len(futures))
		for i, future := range futures {
			errs[i] = errors.Wrapf(future.Get(ctx, nil), "couldn't update node pool %q", input.NodePoolsToUpdate[i].Name)
		}

		if err := errors.Combine(errs...); err != nil {
			return err
		}
	}

	for nodePoolIndex, nodePool := range input.NodePoolsToAdd {
		// Select image for new nodepool (if not specified)
		if nodePool.ImageID == "" {
			activityInput := SelectImageActivityInput{
				ClusterID:    input.ClusterID,
				InstanceType: nodePool.InstanceType,
			}
			var activityOutput SelectImageActivityOutput
			err := workflow.ExecuteActivity(ctx, SelectImageActivityName, activityInput).Get(ctx, &activityOutput)
			if err != nil {
				return err
			}

			input.NodePoolsToAdd[nodePoolIndex].ImageID = activityOutput.ImageID
		}

		// Select volume size
		{
			activityInput := SelectVolumeSizeActivityInput{
				AWSActivityInput: awsActivityInput,
				ImageID:          nodePool.ImageID,
			}
			var activityOutput SelectVolumeSizeActivityOutput
			err := workflow.ExecuteActivity(ctx, SelectVolumeSizeActivityName, activityInput).Get(ctx, &activityOutput)
			if err != nil {
				return err
			}

			if nodePool.VolumeSize == 0 { // Note: not set, using autodefault.
				input.NodePoolsToAdd[nodePoolIndex].VolumeSize = int(math.Max(float64(MinimalVolumeSize), float64(activityOutput.VolumeSize)))
			} else if nodePool.VolumeSize < activityOutput.VolumeSize {
				return errors.Combine(err, errors.Errorf("specified volume size of %dGB for %q is less than the AMI image size of %dGB", nodePool.VolumeSize, nodePool.Name, activityOutput.VolumeSize))
			}
		}
	}

	{
		futures := make([]workflow.Future, len(input.NodePoolsToAdd))

		for i, np := range input.NodePoolsToAdd {
			createWorkerPoolActivityInput := CreateWorkerPoolActivityInput{
				// AWSActivityInput:      awsActivityInput,
				ClusterID:                 input.ClusterID,
				Pool:                      np,
				WorkerInstanceProfile:     PkeGlobalStackName + "-worker-profile",
				VPCID:                     input.VPCID,
				VPCDefaultSecurityGroupID: vpcDefaultSecurityGroupID,
				SubnetIDs:                 input.SubnetIDs,
				ClusterSecurityGroup:      clusterSecurityGroup,
				ExternalBaseUrl:           input.PipelineExternalURL,
				ExternalBaseUrlInsecure:   input.PipelineExternalURLInsecure,
				SSHKeyName:                "pke-ssh-" + input.ClusterName,
			}

			futures[i] = workflow.ExecuteActivity(ctx, CreateWorkerPoolActivityName, createWorkerPoolActivityInput)
		}

		errs := make([]error, len(futures))
		for i, future := range futures {
			errs[i] = errors.Wrapf(future.Get(ctx, nil), "couldn't create node pool %q", input.NodePoolsToAdd[i].Name)
		}

		if err := errors.Combine(errs...); err != nil {
			return err
		}
	}

	return nil
}
