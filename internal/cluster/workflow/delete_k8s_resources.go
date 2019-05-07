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
	"strings"
	"time"

	"github.com/goph/emperror"
	"go.uber.org/cadence/workflow"
)

const DeleteK8sResourcesWorkflowName = "delete-k8s-resources"

type DeleteK8sResourcesWorkflowInput struct {
	OrganizationID uint
	ClusterName    string
	K8sConfig      []byte
}

func DeleteK8sResourcesWorkflow(ctx workflow.Context, input DeleteK8sResourcesWorkflowInput) error {
	logger := workflow.GetLogger(ctx).Sugar()

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// delete all Helm deployments
	{
		activityInput := DeleteHelmDeploymentsActivityInput{
			OrganizationID: input.OrganizationID,
			ClusterName:    input.ClusterName,
			K8sConfig:      input.K8sConfig,
		}
		if err := workflow.ExecuteActivity(ctx, DeleteHelmDeploymentsActivityName, activityInput).Get(ctx, nil); err != nil {
			if strings.Contains(err.Error(), "could not find tiller") {
				logger.Info("could not delete helm deployment because tiller is not running")
			} else {
				return emperror.Wrap(err, "failed to delete Help deployments")
			}
		}
	}

	var deleteUserNamespacesOutput DeleteUserNamespacesActivityOutput

	// delete user namespaces
	{
		activityInput := DeleteUserNamespacesActivityInput{
			OrganizationID: input.OrganizationID,
			ClusterName:    input.ClusterName,
			K8sConfig:      input.K8sConfig,
		}
		if err := workflow.ExecuteActivity(ctx, DeleteUserNamespacesActivityName, activityInput).Get(ctx, &deleteUserNamespacesOutput); err != nil {
			logger.Info(emperror.Wrap(err, "failed to delete user namespaces")) // retry later after resource deletion
		}
	}

	// delete resources in remaining namespaces
	for _, ns := range append(deleteUserNamespacesOutput.NamespacesLeft, "default") {
		activityInput := DeleteNamespaceResourcesActivityInput{
			OrganizationID: input.OrganizationID,
			ClusterName:    input.ClusterName,
			K8sConfig:      input.K8sConfig,
			Namespace:      ns,
		}
		if err := workflow.ExecuteActivity(ctx, DeleteNamespaceResourcesActivityName, activityInput).Get(ctx, nil); err != nil {
			return emperror.Wrapf(err, "failed to delete resources in namespace %q", activityInput.Namespace)
		}
	}

	// delete services in remaining namespaces
	for _, ns := range append(deleteUserNamespacesOutput.NamespacesLeft, "default") {
		activityInput := DeleteNamespaceServicesActivityInput{
			OrganizationID: input.OrganizationID,
			ClusterName:    input.ClusterName,
			K8sConfig:      input.K8sConfig,
			Namespace:      ns,
		}
		if err := workflow.ExecuteActivity(ctx, DeleteNamespaceServicesActivityName, activityInput).Get(ctx, nil); err != nil {
			return emperror.Wrapf(err, "failed to delete services in namespace %q", activityInput.Namespace)
		}
	}

	// delete user namespaces
	{
		activityInput := DeleteUserNamespacesActivityInput{
			K8sConfig: input.K8sConfig,
		}
		if err := workflow.ExecuteActivity(ctx, DeleteUserNamespacesActivityName, activityInput).Get(ctx, nil); err != nil {
			return emperror.Wrap(err, "failed to delete user namespaces")
		}
	}

	return nil
}
