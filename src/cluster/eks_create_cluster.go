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

package cluster

import (
	"time"

	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	eksWorkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
)

const EKSCreateClusterWorkflowName = "eks-create-cluster"

// CreateClusterWorkflowInput holds data needed by the create cluster workflow
type EKSCreateClusterWorkflowInput struct {
	eksWorkflow.CreateInfrastructureWorkflowInput

	OrganizationName string
	PostHooks        pkgCluster.PostHooks
	NodePoolLabels   map[string]map[string]string
}

// CreateClusterWorkflow executes the Cadence workflow responsible for creating and configuring an EKS cluster
func EKSCreateClusterWorkflow(ctx workflow.Context, input EKSCreateClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}
	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: 1 * time.Hour,
		TaskStartToCloseTimeout:      5 * time.Minute,
	}
	ctx = workflow.WithChildOptions(workflow.WithActivityOptions(ctx, ao), cwo)

	infraOutput := eksWorkflow.CreateInfrastructureWorkflowOutput{}
	err := workflow.ExecuteChildWorkflow(
		ctx,
		eksWorkflow.CreateInfraWorkflowName,
		input.CreateInfrastructureWorkflowInput,
	).Get(ctx, &infraOutput)
	if err != nil {
		_ = eksWorkflow.SetClusterErrorStatus(ctx, input.ClusterID, err)
		return err
	}

	{
		activityInput := eksWorkflow.SaveNetworkDetailsInput{
			ClusterID:          input.ClusterID,
			VpcID:              infraOutput.VpcID,
			NodeInstanceRoleID: infraOutput.NodeInstanceRoleID,
			Subnets:            infraOutput.Subnets,
		}
		err := workflow.ExecuteActivity(ctx, eksWorkflow.SaveNetworkDetailsActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			_ = eksWorkflow.SetClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	{
		restoreBackupParams, err := pkgCluster.GetRestoreBackupParams(input.PostHooks)
		if err != nil {
			_ = eksWorkflow.SetClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
		workflowInput := clustersetup.WorkflowInput{
			ConfigSecretID: brn.New(input.OrganizationID, brn.SecretResourceType, infraOutput.ConfigSecretID).String(),
			Cluster: clustersetup.Cluster{
				ID:    input.ClusterID,
				UID:   input.ClusterUID,
				Name:  input.ClusterName,
				Cloud: pkgCluster.Amazon,
			},
			Organization: clustersetup.Organization{
				ID:   input.OrganizationID,
				Name: input.OrganizationName,
			},
			NodePoolLabels:      input.NodePoolLabels,
			RestoreBackupParams: restoreBackupParams,
		}

		future := workflow.ExecuteChildWorkflow(ctx, clustersetup.WorkflowName, workflowInput)
		if err := future.Get(ctx, nil); err != nil {
			_ = eksWorkflow.SetClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	postHookWorkflowInput := RunPostHooksWorkflowInput{
		ClusterID: input.ClusterID,
		PostHooks: BuildWorkflowPostHookFunctions(input.PostHooks, true),
	}

	err = workflow.ExecuteChildWorkflow(ctx, RunPostHooksWorkflowName, postHookWorkflowInput).Get(ctx, nil)
	if err != nil {
		_ = eksWorkflow.SetClusterErrorStatus(ctx, input.ClusterID, err)
		return err
	}

	return nil
}
