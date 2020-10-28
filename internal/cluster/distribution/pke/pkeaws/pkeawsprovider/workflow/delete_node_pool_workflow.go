// Copyright © 2020 Banzai Cloud
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

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/clusterworkflow"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
)

// DeleteNodePoolWorkflowName is the name of the PKE node pool deletion
// workflow.
const DeleteNodePoolWorkflowName = "pke-aws-delete-node-pool"

// DeleteNodePoolWorkflow defines a Cadence workflow encapsulating high level
// input-independent components required to delete an PKE node pool.
type DeleteNodePoolWorkflow struct{}

// DeleteNodePoolWorkflowInput defines the input parameters of an PKE node pool
// deletion.
type DeleteNodePoolWorkflowInput struct {
	ClusterID      uint
	ClusterName    string
	NodePoolName   string
	OrganizationID uint
	Region         string
	SecretID       string

	// Note: ClusterAPI.DeleteCluster, ClusterAPI.UpdateCluster node pool
	// deletions should not change the cluster status (DELETING/UPDATING),
	// because success could not yet mean RUNNING status and errors should be
	// handled by the higher level workflow, but NodePoolAPI.DeleteNodePool node
	// pool deletions should update the cluster status.
	ShouldUpdateClusterStatus bool
}

// NewDeleteNodePoolWorkflow instantiates an EKS node pool deletion workflow.
func NewDeleteNodePoolWorkflow() *DeleteNodePoolWorkflow {
	return &DeleteNodePoolWorkflow{}
}

// Execute runs the workflow.
func (w DeleteNodePoolWorkflow) Execute(ctx workflow.Context, input DeleteNodePoolWorkflowInput) (err error) {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          15 * time.Second,
			BackoffCoefficient:       1.0,
			MaximumAttempts:          30,
			NonRetriableErrorReasons: []string{pkgCadence.ClientErrorReason, "cadenceInternal:Panic"},
		},
	}
	_ctx := ctx
	ctx = workflow.WithActivityOptions(ctx, ao)

	if input.ShouldUpdateClusterStatus {
		defer func() {
			if err == nil {
				return
			}

			_ = clusterworkflow.SetClusterStatus(_ctx, input.ClusterID, cluster.Warning, pkgCadence.UnwrapError(err).Error())
		}()
	}

	{
		activityInput := awsworkflow.DeleteStackActivityInput{
			AWSCommonActivityInput: awsworkflow.AWSCommonActivityInput{
				OrganizationID: input.OrganizationID,
				SecretID:       input.SecretID,
				Region:         input.Region,
				ClusterName:    input.ClusterName,
				AWSClientRequestTokenBase: sdkAmazon.NewNormalizedClientRequestToken(
					workflow.GetInfo(ctx).WorkflowExecution.ID,
				),
			},
			StackID:   "",
			StackName: GenerateNodePoolStackName(input.ClusterName, input.NodePoolName),
		}

		err := workflow.ExecuteActivity(ctx, awsworkflow.DeleteStackActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	{
		activityInput := DeleteStoredNodePoolActivityInput{
			ClusterID:      input.ClusterID,
			ClusterName:    input.ClusterName,
			NodePoolName:   input.NodePoolName,
			OrganizationID: input.OrganizationID,
		}

		err := workflow.ExecuteActivity(ctx, DeleteStoredNodePoolActivityName, activityInput).
			Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	{
		activityInput := clusterworkflow.DeleteNodePoolLabelSetActivityInput{
			ClusterID:    input.ClusterID,
			NodePoolName: input.NodePoolName,
		}

		err := workflow.ExecuteActivity(
			ctx, clusterworkflow.DeleteNodePoolLabelSetActivityName, activityInput,
		).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	if input.ShouldUpdateClusterStatus {
		input := clusterworkflow.SetClusterStatusActivityInput{
			ClusterID:     input.ClusterID,
			Status:        cluster.Running,
			StatusMessage: cluster.RunningMessage,
		}

		err := workflow.ExecuteActivity(ctx, clusterworkflow.SetClusterStatusActivityName, input).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// Register registers the activity in the worker.
func (w DeleteNodePoolWorkflow) Register() {
	workflow.RegisterWithOptions(w.Execute, workflow.RegisterOptions{Name: DeleteNodePoolWorkflowName})
}
