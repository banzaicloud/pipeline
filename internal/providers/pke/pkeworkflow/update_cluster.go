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
	"strings"
	"time"

	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"go.uber.org/cadence/workflow"
)

const UpdateClusterWorkflowName = "pke-update-cluster"

type UpdateClusterWorkflowInput struct {
	OrganizationID      uint
	ClusterID           uint
	ClusterUID          string
	ClusterName         string
	SecretID            string
	Region              string
	PipelineExternalURL string
	NodePools           []NodePool
	VPCID               string
	SubnetIDs           []string
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

	var oldNodePools []NodePool

	if err := workflow.ExecuteActivity(ctx,
		ListNodePoolsActivityName,
		ListNodePoolsActivityInput{
			ClusterID: input.ClusterID,
		}).Get(ctx, &oldNodePools); err != nil {
		return err
	}

	olds := map[string]bool{}
	for _, old := range oldNodePools {
		olds[old.Name] = true
	}

	news := map[string]bool{}
	for _, new := range input.NodePools {
		news[new.Name] = true
	}

	// delete removed nodepools
	for _, np := range oldNodePools {
		if news[np.Name] || np.Master || !np.Worker {
			continue
		}

		err := workflow.ExecuteActivity(ctx, DeletePoolActivityName, DeletePoolActivityInput{
			//AWSActivityInput: awsActivityInput,
			ClusterID: input.ClusterID,
			Pool:      np,
		}).Get(ctx, nil)
		if err != nil {
			return emperror.Wrapf(err, "deleting %q", np.Name)
		}
	}

	// create/change nodepools that are not removed
	for _, np := range input.NodePools {
		if np.Master || np.Name == "master" {
			continue
		}

		if olds[np.Name] { // update existing

			stackName := fmt.Sprintf("pke-pool-%s-worker-%s", input.ClusterName, np.Name)
			var cfOut map[string]string

			err := workflow.ExecuteActivity(ctx,
				WaitCFCompletionActivityName,
				WaitCFCompletionActivityInput{
					AWSActivityInput: awsActivityInput,
					StackID:          stackName}).Get(ctx, &cfOut)
			if err != nil {
				return emperror.Wrap(err, fmt.Sprintf("can't find AutoScalingGroup for pool %q", np.Name))
			}

			asg, ok := cfOut["AutoScalingGroupId"]
			if !ok {
				return errors.New(fmt.Sprintf("can't find AutoScalingGroup for pool %q", np.Name))
			}

			err = workflow.ExecuteActivity(ctx,
				UpdatePoolActivityName,
				UpdatePoolActivityInput{
					AWSActivityInput: awsActivityInput,
					Pool:             np,
					AutoScalingGroup: asg,
				}).Get(ctx, nil)
			if err != nil {
				return err
			}

			continue
		}

		// create new

		createWorkerPoolActivityInput := CreateWorkerPoolActivityInput{
			//AWSActivityInput:      awsActivityInput,
			ClusterID:             input.ClusterID,
			Pool:                  np,
			WorkerInstanceProfile: PkeGlobalStackName + "-worker-profile",
			VPCID:                 input.VPCID,
			SubnetID:              strings.Join(input.SubnetIDs, ","),
			ClusterSecurityGroup:  clusterSecurityGroup,
			ExternalBaseUrl:       input.PipelineExternalURL,
			SSHKeyName:            "pke-ssh-" + input.ClusterName,
		}

		err := workflow.ExecuteActivity(ctx, CreateWorkerPoolActivityName, createWorkerPoolActivityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
