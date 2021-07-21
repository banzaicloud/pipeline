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

package eksworkflow

import (
	"time"

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	eksWorkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

const UpdateClusterWorkflowName = "eks-update-cluster-v2"

// UpdateClusterWorkflowInput holds data needed to update EKS cluster version
type UpdateClusterWorkflowInput struct {
	Region           string
	OrganizationID   uint
	ProviderSecretID string
	ConfigSecretID   string

	ClusterID   uint
	ClusterName string

	Version string
}

type UpdateClusterWorkflow struct {
}

func NewUpdateClusterWorkflow() UpdateClusterWorkflow {
	return UpdateClusterWorkflow{}
}

// Register registers the activity in the worker.
func (w UpdateClusterWorkflow) Register(worker worker.Registry) {
	worker.RegisterWorkflowWithOptions(w.Execute, workflow.RegisterOptions{Name: UpdateClusterWorkflowName})
}

// Execute executes the Cadence workflow responsible for updating EKS cluster version
func (w UpdateClusterWorkflow) Execute(ctx workflow.Context, input UpdateClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic", eksWorkflow.ErrReasonStackFailed},
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// execute the version update
	var updateOutput UpdateClusterVersionActivityOutput
	{
		activityInput := UpdateClusterVersionActivityInput{
			OrganizationID:   input.OrganizationID,
			ProviderSecretID: input.ProviderSecretID,
			Region:           input.Region,
			ClusterName:      input.ClusterName,
			Version:          input.Version,
		}
		err := workflow.ExecuteActivity(ctx, UpdateClusterVersionActivityName, activityInput).Get(ctx, &updateOutput)
		if err != nil {
			_ = eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error())
			return err
		}
	}

	// wait for the update to finish
	{
		activityInput := &WaitUpdateClusterVersionActivityInput{
			OrganizationID:   input.OrganizationID,
			ProviderSecretID: input.ProviderSecretID,
			Region:           input.Region,
			ClusterName:      input.ClusterName,
			UpdateID:         updateOutput.UpdateID,
		}

		ctx := workflow.WithStartToCloseTimeout(ctx, 2*time.Hour)

		err := workflow.ExecuteActivity(ctx, WaitUpdateClusterVersionActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			_ = eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error())
			return err
		}
	}

	// save the new cluster version to db
	{
		activityInput := &eksWorkflow.SaveClusterVersionActivityInput{
			ClusterID: input.ClusterID,
			Version:   input.Version,
		}
		err := workflow.ExecuteActivity(ctx, eksWorkflow.SaveClusterVersionActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			_ = eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error())
			return err
		}
	}

	_ = eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Running, pkgCluster.RunningMessage)

	return nil
}
