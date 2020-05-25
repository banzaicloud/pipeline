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

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
)

const DeleteNodePoolWorkflowName = "pke-vsphere-delete-nodepool"

// DeleteNodePoolWorkflowInput
type DeleteNodePoolWorkflowInput struct {
	ClusterID      uint
	ClusterName    string
	OrganizationID uint
	SecretID       string
	K8sSecretID    string
	NodePool       NodePool
}

type NodePool struct {
	Name string
	Size int
}

func DeleteNodePoolWorkflow(ctx workflow.Context, input DeleteNodePoolWorkflowInput) error {
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

	nodeNamesToDelete := make([]string, 0)

	for j := 1; j <= input.NodePool.Size; j++ {
		nodeNamesToDelete = append(nodeNamesToDelete, pke.GetVMName(input.ClusterName, input.NodePool.Name, j))
	}

	// Delete k8s nodes
	{
		futures := make(map[string]workflow.Future)

		for _, nodeName := range nodeNamesToDelete {
			activityInput := DeleteK8sNodeActivityInput{
				OrganizationID: input.OrganizationID,
				ClusterName:    input.ClusterName,
				K8sSecretID:    input.K8sSecretID,
				Name:           nodeName,
			}

			futures[nodeName] = workflow.ExecuteActivity(ctx, DeleteK8sNodeActivityName, activityInput)
		}

		errs := []error{}

		for i := range futures {
			errs = append(errs, errors.WrapIff(futures[i].Get(ctx, nil), "deleting kubernetes node %q", i))
		}

		if err := errors.Combine(errs...); err != nil {
			return err
		}
	}

	// Delete VM's
	{
		futures := make(map[string]workflow.Future)

		for _, nodeName := range nodeNamesToDelete {
			activityInput := DeleteNodeActivityInput{
				OrganizationID: input.OrganizationID,
				SecretID:       input.SecretID,
				ClusterID:      input.ClusterID,
				ClusterName:    input.ClusterName,
				Node: Node{
					Name: nodeName,
				},
			}

			futures[nodeName] = workflow.ExecuteActivity(ctx, DeleteNodeActivityName, activityInput)
		}

		errs := []error{}

		for i := range futures {
			errs = append(errs, errors.WrapIff(futures[i].Get(ctx, nil), "deleting node %q", i))
		}

		if err := errors.Combine(errs...); err != nil {
			return err
		}
	}

	// delete node pool from data store
	{
		activityInput := DeleteNodePoolFromStoreActivityInput{
			ClusterID:    input.ClusterID,
			NodePoolName: input.NodePool.Name,
		}
		err := workflow.ExecuteActivity(ctx, DeleteNodePoolFromStoreActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
