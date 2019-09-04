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
	"fmt"
	"time"

	intClusterWorkflow "github.com/banzaicloud/pipeline/internal/cluster/workflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"

	"go.uber.org/cadence/workflow"
)

const DeleteClusterWorkflowName = "pke-azure-delete-cluster"

type DeleteClusterWorkflowInput struct {
	OrganizationID       uint
	SecretID             string
	ClusterID            uint
	ClusterName          string
	ClusterUID           string
	K8sSecretID          string
	ResourceGroupName    string
	LoadBalancerName     string
	PublicIPAddressNames []string
	RouteTableName       string
	ScaleSetNames        []string
	SecurityGroupNames   []string
	VirtualNetworkName   string

	Forced bool
}

func DeleteClusterWorkflow(ctx workflow.Context, input DeleteClusterWorkflowInput) error {

	logger := workflow.GetLogger(ctx).Sugar()

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}
	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: 30 * time.Minute,
		TaskStartToCloseTimeout:      40 * time.Minute,
	}
	ctx = workflow.WithChildOptions(workflow.WithActivityOptions(ctx, ao), cwo)

	// delete k8s resources
	if input.K8sSecretID != "" {
		wfInput := intClusterWorkflow.DeleteK8sResourcesWorkflowInput{
			OrganizationID: input.OrganizationID,
			ClusterName:    input.ClusterName,
			K8sSecretID:    input.K8sSecretID,
		}
		if err := workflow.ExecuteChildWorkflow(ctx, intClusterWorkflow.DeleteK8sResourcesWorkflowName, wfInput).Get(ctx, nil); err != nil {
			if input.Forced {
				logger.Errorw("deleting k8s resources failed", "error", err)
			} else {
				_ = setClusterErrorStatus(ctx, input.ClusterID, err)
				return err
			}
		}
	}

	// clean up DNS records
	{
		activityInput := intClusterWorkflow.DeleteClusterDNSRecordsActivityInput{
			OrganizationID: input.OrganizationID,
			ClusterUID:     input.ClusterUID,
		}
		if err := workflow.ExecuteActivity(ctx, intClusterWorkflow.DeleteClusterDNSRecordsActivityName, activityInput).Get(ctx, nil); err != nil {
			if input.Forced {
				logger.Errorw("deleting cluster DNS records failed", "error", err)
			} else {
				_ = setClusterErrorStatus(ctx, input.ClusterID, err)
				return err
			}
		}
	}

	// delete infra
	{
		infraInput := DeleteAzureInfrastructureWorkflowInput{
			OrganizationID:       input.OrganizationID,
			SecretID:             input.SecretID,
			ClusterName:          input.ClusterName,
			ResourceGroupName:    input.ResourceGroupName,
			LoadBalancerName:     input.LoadBalancerName,
			PublicIPAddressNames: input.PublicIPAddressNames,
			RouteTableName:       input.RouteTableName,
			ScaleSetNames:        input.ScaleSetNames,
			SecurityGroupNames:   input.SecurityGroupNames,
			VirtualNetworkName:   input.VirtualNetworkName,
		}
		err := workflow.ExecuteChildWorkflow(ctx, DeleteInfraWorkflowName, infraInput).Get(ctx, nil)
		if err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	// delete unused secrets
	{
		activityInput := intClusterWorkflow.DeleteUnusedClusterSecretsActivityInput{
			OrganizationID: input.OrganizationID,
			ClusterUID:     input.ClusterUID,
		}
		if err := workflow.ExecuteActivity(ctx, intClusterWorkflow.DeleteUnusedClusterSecretsActivityName, activityInput).Get(ctx, nil); err != nil {
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, fmt.Sprintf("failed to delete unused cluster secrets: %v", err)) // nolint: errcheck
		}
	}

	// remove dex client (if we created it)
	{
		deleteDexClientActivityInput := &pkeworkflow.DeleteDexClientActivityInput{
			ClusterID: input.ClusterID,
		}
		if err := workflow.ExecuteActivity(ctx, pkeworkflow.DeleteDexClientActivityName, deleteDexClientActivityInput).Get(ctx, nil); err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	// delete cluster from data store
	{
		activityInput := DeleteClusterFromStoreActivityInput{
			ClusterID: input.ClusterID,
		}
		err := workflow.ExecuteActivity(ctx, DeleteClusterFromStoreActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	return nil
}
