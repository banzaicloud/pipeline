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

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	intClusterWorkflow "github.com/banzaicloud/pipeline/internal/cluster/workflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

const DeleteClusterWorkflowName = "pke-vsphere-delete-cluster"

// DeleteClusterWorkflowInput
type DeleteClusterWorkflowInput struct {
	ClusterID        uint
	ClusterName      string
	ClusterUID       string
	K8sSecretID      string
	OrganizationID   uint
	OrganizationName string
	ResourcePoolName string
	FolderName       string
	DatastoreName    string
	SecretID         string
	MasterNodes      []Node
	Nodes            []Node
	Forced           bool
}

func DeleteClusterWorkflow(ctx workflow.Context, input DeleteClusterWorkflowInput) error {
	logger := workflow.GetLogger(ctx).Sugar()

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

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

	// Delete VM nodes
	{
		for _, node := range input.Nodes {
			activityInput := DeleteNodeActivityInput{
				OrganizationID: input.OrganizationID,
				SecretID:       input.SecretID,
				ClusterID:      input.ClusterID,
				ClusterName:    input.ClusterName,
				Node:           node,
			}

			err := workflow.ExecuteActivity(ctx, DeleteNodeActivityName, activityInput).Get(ctx, nil)
			if err != nil {
				if input.Forced {
					logger.Errorw("delete node failed", "error", err, "node", node.Name)
				} else {
					e := errors.WrapIff(err, "deleting node %q", node.Name)
					_ = setClusterErrorStatus(ctx, input.ClusterID, e)
					return e
				}
			}
		}
	}

	// Delete master VM nodes
	{
		futures := make(map[string]workflow.Future)

		for _, node := range input.MasterNodes {
			activityInput := DeleteNodeActivityInput{
				OrganizationID: input.OrganizationID,
				SecretID:       input.SecretID,
				ClusterID:      input.ClusterID,
				ClusterName:    input.ClusterName,
				Node:           node,
			}
			futures[node.Name] = workflow.ExecuteActivity(ctx, DeleteNodeActivityName, activityInput)
		}

		errs := []error{}

		for i := range futures {
			var existed bool
			errs = append(errs, errors.WrapIff(futures[i].Get(ctx, &existed), "deleting node %q", i))
		}

		if err := errors.Combine(errs...); err != nil {
			if input.Forced {
				logger.Errorw("delete master node failed", "error", err)
			} else {
				_ = setClusterErrorStatus(ctx, input.ClusterID, err)
				return err
			}
		}
	}

	// delete unused secrets
	{
		activityInput := intClusterWorkflow.DeleteUnusedClusterSecretsActivityInput{
			OrganizationID: input.OrganizationID,
			ClusterUID:     input.ClusterUID,
		}
		if err := workflow.ExecuteActivity(ctx, intClusterWorkflow.DeleteUnusedClusterSecretsActivityName, activityInput).Get(ctx, nil); err != nil {
			_ = setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, fmt.Sprintf("failed to delete unused cluster secrets: %v", pkgCadence.UnwrapError(err)))
		}
	}

	// remove dex client (if we created it)
	{
		deleteDexClientActivityInput := &pkeworkflow.DeleteDexClientActivityInput{
			ClusterID: input.ClusterID,
		}
		if err := workflow.ExecuteActivity(ctx, pkeworkflow.DeleteDexClientActivityName, deleteDexClientActivityInput).Get(ctx, nil); err != nil {
			if input.Forced {
				logger.Errorw("delete dex client failed", "error", err)
			} else {
				_ = setClusterErrorStatus(ctx, input.ClusterID, err)
				return err
			}
		}
	}

	// delete cluster from data store
	{
		activityInput := DeleteClusterFromStoreActivityInput{
			ClusterID: input.ClusterID,
		}
		err := workflow.ExecuteActivity(ctx, DeleteClusterFromStoreActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			if input.Forced {
				logger.Errorw("delete cluster from data store", "error", err)
			} else {
				_ = setClusterErrorStatus(ctx, input.ClusterID, err)
				return err
			}
		}
	}

	return nil
}
