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
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	pkgcadence "github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
	"github.com/banzaicloud/pipeline/pkg/cluster"
)

// CreateNodePoolsWorkflowName is the name of the EKS workflow creating new node
// pools in a cluster.
const CreateNodePoolsWorkflowName = "eks-create-node-pools"

// CreateNodePoolsWorkflow defines a Cadence workflow encapsulating high level
// input-independent components required to create multiple EKS node pools.
type CreateNodePoolsWorkflow struct{}

// CreateNodePoolsWorkflowInput defines the input parameters of an EKS node pool
// creation.
type CreateNodePoolsWorkflowInput struct {
	ClusterID                    uint
	CreatorUserID                uint
	NodePools                    map[string]eks.NewNodePool
	NodePoolSubnetIDs            map[string][]string
	ShouldCreateNodePoolLabelSet bool
	ShouldStoreNodePool          bool
	ShouldUpdateClusterStatus    bool
}

// NewCreateNodePoolsWorkflow instantiates an EKS node pools creation workflow.
func NewCreateNodePoolsWorkflow() *CreateNodePoolsWorkflow {
	return &CreateNodePoolsWorkflow{}
}

// Execute runs the workflow.
func (w CreateNodePoolsWorkflow) Execute(ctx workflow.Context, input CreateNodePoolsWorkflowInput) (err error) {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          15 * time.Second,
			BackoffCoefficient:       1.0,
			MaximumAttempts:          30,
			NonRetriableErrorReasons: []string{pkgcadence.ClientErrorReason, "cadenceInternal:Panic"},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	if input.ShouldUpdateClusterStatus {
		defer func() { // Note: update cluster status on error.
			if err != nil {
				err = pkgcadence.UnwrapError(err)
				_ = SetClusterStatus(ctx, input.ClusterID, cluster.Warning, err.Error())
			}
		}()
	}

	createNodePoolFutures := make(map[string]workflow.Future, len(input.NodePools))
	for nodePoolName, nodePool := range input.NodePools {
		input := CreateNodePoolWorkflowInput{
			ClusterID:                    input.ClusterID,
			NodePool:                     nodePool,
			NodePoolSubnetIDs:            input.NodePoolSubnetIDs[nodePoolName],
			ShouldCreateNodePoolLabelSet: input.ShouldCreateNodePoolLabelSet, // Note: depends on executing context/workflow (example: ClusterAPI.EKS.CreateCluster->false, ClusterAPI.EKS.NodePoolAPI.CreateNodePool->true).
			ShouldStoreNodePool:          input.ShouldStoreNodePool,          // Note: depends on executing context/workflow (example: ClusterAPI.EKS.CreateCluster->false, ClusterAPI.EKS.NodePoolAPI.CreateNodePool->true).
			ShouldUpdateClusterStatus:    false,                              // Note: Status is either handled above (example: ClusterAPI.EKS.CreateCluster) or in this aggregate workflow (example: ClusterAPI.EKS.NodePoolAPI.CreateNodePool), not in the individual CreateNodePoolWorkflow workflows.
			CreatorUserID:                input.CreatorUserID,
		}

		createNodePoolFutures[nodePoolName] = workflow.ExecuteChildWorkflow(ctx, CreateNodePoolWorkflowName, input)
	}

	createNodePoolErrors := make([]error, 0, len(input.NodePools))
	for nodePoolName, future := range createNodePoolFutures {
		err = pkgcadence.UnwrapError(future.Get(ctx, nil))
		if err != nil {
			createNodePoolErrors = append(
				createNodePoolErrors,
				errors.Wrapf(err, "creating node pool failed, nodePool: %s", nodePoolName),
			)
		}
	}
	if len(createNodePoolErrors) != 0 {
		return errors.Combine(createNodePoolErrors...)
	}

	if input.ShouldUpdateClusterStatus {
		err = SetClusterStatus(ctx, input.ClusterID, cluster.Running, cluster.RunningMessage)
		if err != nil {
			return err
		}
	}

	return nil
}

// Register registers the activity in the worker.
func (w CreateNodePoolsWorkflow) Register(worker worker.Registry) {
	worker.RegisterWorkflowWithOptions(w.Execute, workflow.RegisterOptions{Name: CreateNodePoolsWorkflowName})
}
