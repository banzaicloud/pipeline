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

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	intClusterWorkflow "github.com/banzaicloud/pipeline/internal/cluster/workflow"
	eksWorkflow "github.com/banzaicloud/pipeline/internal/providers/amazon/eks/workflow"
)

const EKSDeleteClusterWorkflowName = "eks-delete-cluster"

// DeleteClusterWorkflowInput holds data needed by the delete cluster workflow
type EKSDeleteClusterWorkflowInput struct {
	OrganizationID uint
	SecretID       string
	Region         string

	ClusterName   string
	NodePoolNames []string

	ClusterID  uint
	ClusterUID string

	// the identifier of the kubeconfig secret of the cluster
	K8sSecretID string
	DefaultUser bool

	// force delete
	Forced bool

	GeneratedSSHKeyUsed bool
}

// DeleteClusterWorkflow executes the Cadence workflow responsible for deleting an EKS cluster
func EKSDeleteClusterWorkflow(ctx workflow.Context, input EKSDeleteClusterWorkflowInput) error {
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
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic", eksWorkflow.ErrReasonStackFailed},
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
					_ = eksWorkflow.SetClusterErrorStatus(ctx, input.ClusterID, err)
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
				_ = eksWorkflow.SetClusterErrorStatus(ctx, input.ClusterID, err)
				return err
			}
		}
	}

	// delete infra child workflow
	{
		infraInput := eksWorkflow.DeleteInfrastructureWorkflowInput{
			OrganizationID:   input.OrganizationID,
			SecretID:         input.SecretID,
			Region:           input.Region,
			ClusterUID:       input.ClusterUID,
			ClusterName:      input.ClusterName,
			NodePoolNames:    input.NodePoolNames,
			DefaultUser:      input.DefaultUser,
			GeneratedSSHUsed: input.GeneratedSSHKeyUsed,
		}

		err := workflow.ExecuteChildWorkflow(ctx, eksWorkflow.DeleteInfraWorkflowName, infraInput).Get(ctx, nil)
		if err != nil {
			if input.Forced {
				logger.Errorw("deleting cluster infrastructure failed", "error", err)
			} else {
				_ = eksWorkflow.SetClusterErrorStatus(ctx, input.ClusterID, err)
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
				_ = eksWorkflow.SetClusterErrorStatus(ctx, input.ClusterID, err)
				return err
			}
		}
	}

	// delete cluster from data store
	{
		activityInput := eksWorkflow.DeleteClusterFromStoreActivityInput{
			ClusterID: input.ClusterID,
		}
		err := workflow.ExecuteActivity(ctx, eksWorkflow.DeleteClusterFromStoreActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			_ = eksWorkflow.SetClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	return nil
}
