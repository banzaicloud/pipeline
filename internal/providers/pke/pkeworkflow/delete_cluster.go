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
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
)

const DeleteClusterWorkflowName = "pke-delete-cluster"

type DeleteClusterWorkflowInput struct {
	ClusterID uint
}

func DeleteClusterWorkflow(ctx workflow.Context, input DeleteClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 1.5,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	var nodePools []NodePool
	listNodePoolsActivityInput := ListNodePoolsActivityInput{
		ClusterID: input.ClusterID,
	}

	var err error
	if err = workflow.ExecuteActivity(ctx, ListNodePoolsActivityName, listNodePoolsActivityInput).Get(ctx, &nodePools); err != nil {
		return err
	}

	// terminate worker nodes
	{
		futures := make([]workflow.Future, 0, 2*len(nodePools))
		for _, np := range nodePools {
			if !np.Master && np.Worker {
				deletePoolActivityInput := DeletePoolActivityInput{
					ClusterID: input.ClusterID,
					Pool:      np,
				}

				// initiate deletion
				futures = append(futures, workflow.ExecuteActivity(ctx, DeletePoolActivityName, deletePoolActivityInput))

				// set the heartbeat timeout!
				// initiate wait for deletion to complete
				futures = append(futures, workflow.ExecuteActivity(workflow.WithHeartbeatTimeout(ctx, 1*time.Minute),
					WaitForDeletePoolActivityName,
					deletePoolActivityInput))
			}
		}

		errs := make([]error, len(futures))
		for _, future := range futures {
			if future != nil {
				if e := future.Get(ctx, nil); e != nil {
					errs = append(errs, errors.Wrapf(e, "couldn't terminate master node pool"))
				}
			}
		}

		if err := errors.Combine(errs...); err != nil {
			return err
		}

	}

	// release NLB

	deleteNLBActivityInput := &DeleteNLBActivityInput{
		ClusterID: input.ClusterID,
	}
	if err = workflow.ExecuteActivity(ctx, DeleteNLBActivityName, deleteNLBActivityInput).Get(ctx, nil); err != nil {
		return err
	}
	if err = workflow.ExecuteActivity(
		workflow.WithHeartbeatTimeout(ctx, 1*time.Minute),
		WaitForDeleteNLBActivityName,
		deleteNLBActivityInput).Get(ctx, nil); err != nil {
		return err
	}

	// terminate master nodes
	{
		futures := make([]workflow.Future, 0, 2*len(nodePools))

		for _, np := range nodePools {
			if np.Master || !np.Worker {
				deletePoolActivityInput := DeletePoolActivityInput{
					ClusterID: input.ClusterID,
					Pool:      np,
				}

				// initiate deletion
				futures = append(futures, workflow.ExecuteActivity(ctx, DeletePoolActivityName, deletePoolActivityInput))

				// set the heartbeat timeout!
				// initiate wait for deletion to complete
				futures = append(futures, workflow.ExecuteActivity(workflow.WithHeartbeatTimeout(ctx, 1*time.Minute),
					WaitForDeletePoolActivityName,
					deletePoolActivityInput))
			}
		}

		errs := make([]error, len(futures))
		for _, future := range futures {
			if future != nil {
				if e := future.Get(ctx, nil); e != nil {
					errs = append(errs, errors.Wrapf(e, "couldn't terminate worker node pool"))
				}
			}
		}

		if err := errors.Combine(errs...); err != nil {
			return err
		}
	}

	// clean-up ssh key
	deleteSSHKeyPairActivityInput := &DeleteSSHKeyPairActivityInput{
		ClusterID: input.ClusterID,
	}
	if err := workflow.ExecuteActivity(ctx, DeleteSSHKeyPairActivityName, deleteSSHKeyPairActivityInput).Get(ctx, nil); err != nil {
		return err
	}

	// release elastic ip

	deleteElasticIPActivityInput := &DeleteElasticIPActivityInput{
		ClusterID: input.ClusterID,
	}
	if err = workflow.ExecuteActivity(ctx, DeleteElasticIPActivityName, deleteElasticIPActivityInput).Get(ctx, nil); err != nil {
		return err
	}

	// remove vpc (if we created it)

	deleteVPCActivityInput := &DeleteVPCActivityInput{
		ClusterID: input.ClusterID,
	}
	if err = workflow.ExecuteActivity(ctx, DeleteVPCActivityName, deleteVPCActivityInput).Get(ctx, nil); err != nil {
		return err
	}

	if err = workflow.ExecuteActivity(workflow.WithHeartbeatTimeout(ctx, 1*time.Minute), WaitForDeleteVPCActivityName, deleteVPCActivityInput).Get(ctx, nil); err != nil {
		return err
	}

	// remove dex client (if we created it)

	deleteDexClientActivityInput := &DeleteDexClientActivityInput{
		ClusterID: input.ClusterID,
	}
	if err = workflow.ExecuteActivity(ctx, DeleteDexClientActivityName, deleteDexClientActivityInput).Get(ctx, nil); err != nil {
		return err
	}

	// TODO: remove roles (probably not needed)

	return nil
}
