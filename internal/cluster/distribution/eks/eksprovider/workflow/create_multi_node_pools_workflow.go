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

// CreateMultiNodePoolsWorkflowName is the name of the EKS workflow creating a new
// node pool in a cluster.
const CreateMultiNodePoolsWorkflowName = "eks-create-multi-node-pools"

// CreateMultiNodePoolsWorkflow defines a Cadence workflow encapsulating high level
// input-independent components required to create multiple EKS node pools.
type CreateMultiNodePoolsWorkflow struct{}

// CreateMultiNodePoolsWorkflowInput defines the input parameters of an EKS node pool
// creation.
type CreateMultiNodePoolsWorkflowInput struct {
	ClusterID     uint
	CreatorUserID uint
	NodePoolList  []eks.NewNodePool
}

// NewCreateMultiNodePoolWorkflow instantiates an EKS node pool creation workflow.
func NewCreateMultiNodePoolsWorkflow() *CreateMultiNodePoolsWorkflow {
	return &CreateMultiNodePoolsWorkflow{}
}

// Execute runs the workflow.
func (w CreateMultiNodePoolsWorkflow) Execute(ctx workflow.Context, input CreateMultiNodePoolsWorkflowInput) (err error) {
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

	createNodePoolFutures := make([]workflow.Future, 0, len(input.NodePoolList))
	for _, nodePool := range input.NodePoolList {
		input := CreateNodePoolWorkflowInput{
			ClusterID:                    input.ClusterID,
			NodePool:                     nodePool,
			NodePoolSubnetIDs:            []string{nodePool.SubnetID},
			ShouldCreateNodePoolLabelSet: true,
			ShouldStoreNodePool:          true,
			ShouldUpdateClusterStatus:    false,
			CreatorUserID:                input.CreatorUserID,
		}

		createNodePoolFutures = append(createNodePoolFutures, workflow.ExecuteChildWorkflow(ctx, CreateNodePoolWorkflowName, input))
	}

	createNodePoolErrors := make([]error, 0, len(input.NodePoolList))
	for _, future := range createNodePoolFutures {
		createNodePoolErrors = append(createNodePoolErrors, pkgcadence.UnwrapError(future.Get(ctx, nil)))
	}
	if err := errors.Combine(createNodePoolErrors...); err != nil {
		// TODO should we set Warning state for cluster in this case?
		return err
	}

	err = SetClusterStatus(ctx, input.ClusterID, cluster.Running, cluster.RunningMessage)
	if err != nil {
		return err
	}

	return nil
}

// Register registers the activity in the worker.
func (w CreateMultiNodePoolsWorkflow) Register(worker worker.Registry) {
	worker.RegisterWorkflowWithOptions(w.Execute, workflow.RegisterOptions{Name: CreateMultiNodePoolsWorkflowName})
}
