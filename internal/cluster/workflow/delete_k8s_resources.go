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
	"github.com/goph/emperror"
	"go.uber.org/cadence/workflow"
)

const DeleteK8sResourcesWorkflowName = "delete-k8s-resources"

type DeleteK8sResourcesWorkflowInput struct {
	K8sConfig []byte
}

func DeleteK8sResourcesWorkflow(ctx workflow.Context, input DeleteK8sResourcesWorkflowInput) error {
	// delete all Helm deployments
	{
		activityInput := DeleteHelmDeploymentsActivityInput{
			K8sConfig: input.K8sConfig,
		}
		if err := workflow.ExecuteActivity(ctx, DeleteHelmDeploymentsActivityName, activityInput).Get(ctx, nil); err != nil {
			return emperror.Wrap(err, "failed to delete Help deployments")
		}
	}
	{
		/*
				err = deleteAllResources(config, logger)
			   	if err != nil {
			   		err = emperror.Wrap(err, "failed to delete Kubernetes resources")

			   		if !force {
			   			cluster.SetStatus(pkgCluster.Error, err.Error())

			   			return err
			   		}

			   		logger.Error(err)
			   	}
		*/
	}
	return nil
}
