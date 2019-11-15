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

	intClusterWorkflow "github.com/banzaicloud/pipeline/internal/cluster/workflow"
)

const DeleteClusterWorkflowName = "eks-delete-cluster"

// DeleteClusterWorkflowInput holds data needed by the delete cluster workflow
type DeleteClusterWorkflowInput struct {
	OrganizationID uint
	SecretID       string
	Region         string

	ClusterName string

	ClusterID  uint
	ClusterUID string

	// the identifier of the kubeconfig secret of the cluster
	K8sSecretID string

	// force delete
	Forced bool
}

// DeleteClusterWorkflow executes the Cadence workflow responsible for deleting an EKS cluster
func DeleteClusterWorkflow(ctx workflow.Context, input DeleteClusterWorkflowInput) error {
	logger := workflow.GetLogger(ctx).Sugar()

	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: 1 * time.Hour,
		TaskStartToCloseTimeout:      5 * time.Minute,
	}

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic", ErrReasonStackFailed},
		},
	}

	ctx = workflow.WithChildOptions(ctx, cwo)
	ctx = workflow.WithActivityOptions(ctx, ao)

	// delete K8s resources
	{
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
					if cadence.IsCustomError(err) {
						cerr := err.(*cadence.CustomError)
						if cerr.HasDetails() {
							var errDetails string
							if err = errors.WrapIf(cerr.Details(&errDetails), "couldn't get error details that caused Kubernetes resources deletion to fail"); err != nil {
								return err
							}

							return errors.New(errDetails)
						}
					}
					return err
				}
			}
		}
	}

	// delete cluster DNS records
	{
		activityInput := intClusterWorkflow.DeleteClusterDNSRecordsActivityInput{
			OrganizationID: input.OrganizationID,
			ClusterUID:     input.ClusterUID,
		}
		if err := workflow.ExecuteActivity(ctx, intClusterWorkflow.DeleteClusterDNSRecordsActivityName, activityInput).Get(ctx, nil); err != nil {
			if input.Forced {
				logger.Errorw("deleting cluster DNS records failed", "error", err)
			} else {
				if cadence.IsCustomError(err) {
					cerr := err.(*cadence.CustomError)
					if cerr.HasDetails() {
						var errDetails string
						if err = errors.WrapIf(cerr.Details(&errDetails), "couldn't get error details that caused cluster DNS records deletion to fail"); err != nil {
							return err
						}

						return errors.New(errDetails)
					}
				}
				return err
			}
		}
	}

	// delete infra child workflow
	{
		infraInput := DeleteInfrastructureWorkflowInput{
			OrganizationID: input.OrganizationID,
			SecretID:       input.SecretID,
			Region:         input.Region,
			ClusterName:    input.ClusterName,
		}

		err := workflow.ExecuteChildWorkflow(ctx, DeleteInfraWorkflowName, infraInput).Get(ctx, nil)
		if err != nil {
			if input.Forced {
				logger.Errorw("deleting cluster infrastructure failed", "error", err)
			} else {
				if cadence.IsCustomError(err) {
					cerr := err.(*cadence.CustomError)
					if cerr.HasDetails() {
						var errDetails string
						if err = errors.WrapIf(cerr.Details(&errDetails), "couldn't get error details that caused the deletion of cluster infrastructure to fail"); err != nil {
							return err
						}

						return errors.New(errDetails)
					}
				}
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
			if input.Forced {
				logger.Errorw("failed to delete unused cluster secrets", "error", err)
			} else {
				if cadence.IsCustomError(err) {
					cerr := err.(*cadence.CustomError)
					if cerr.HasDetails() {
						var errDetails string
						if err = errors.WrapIf(cerr.Details(&errDetails), "couldn't get error details that caused the deletion of unused cluster secrets to fail"); err != nil {
							return err
						}

						return errors.New(errDetails)
					}
				}
				return err
			}
		}
	}

	//TODO: DeleteClusterFromStoreActivityName  child workflow

	return nil
}
