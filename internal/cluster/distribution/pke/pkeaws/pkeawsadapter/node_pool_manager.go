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

package pkeawsadapter

import (
	"context"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsworkflow"
)

type nodePoolManager struct {
	enterprise     bool
	namespace      string
	workflowClient client.Client
}

// NewNodePoolManager returns a new pke.NodePoolManager
// that manages node pools asynchronously via Cadence workflows.
func NewNodePoolManager(
	enterprise bool,
	namespace string,
	workflowClient client.Client,
) pke.NodePoolManager {
	return nodePoolManager{
		enterprise:     enterprise,
		namespace:      namespace,
		workflowClient: workflowClient,
	}
}

// DeleteNodePool deletes an existing node pool in a cluster.
func (n nodePoolManager) DeleteNodePool(
	ctx context.Context, c cluster.Cluster, existingNodePool pke.ExistingNodePool, shouldUpdateClusterStatus bool,
) (err error) {
	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 30 * 24 * 60 * time.Minute,
	}

	input := workflow.DeleteNodePoolWorkflowInput{
		ClusterID:                 c.ID,
		ClusterName:               c.Name,
		NodePoolName:              existingNodePool.Name,
		OrganizationID:            c.OrganizationID,
		Region:                    c.Location,
		SecretID:                  c.SecretID.ResourceID,
		ShouldUpdateClusterStatus: shouldUpdateClusterStatus,
	}

	_, err = n.workflowClient.StartWorkflow(ctx, workflowOptions, workflow.DeleteNodePoolWorkflowName, input)
	if err != nil {
		return errors.WrapWithDetails(err, "failed to start workflow", "workflow", workflow.DeleteNodePoolWorkflowName)
	}

	return nil
}

func (n nodePoolManager) UpdateNodePool(
	ctx context.Context,
	c cluster.Cluster,
	nodePoolName string,
	nodePoolUpdate pke.NodePoolUpdate,
) (string, error) {
	taskList := "pipeline"
	if n.enterprise {
		taskList = "pipeline-enterprise"
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     taskList,
		ExecutionStartToCloseTimeout: 30 * 24 * 60 * time.Minute,
	}

	input := pkeawsworkflow.UpdateNodePoolWorkflowInput{
		ProviderSecretID: c.SecretID.String(),
		Region:           c.Location,

		StackName: pkeaws.GenerateNodePoolStackName(c.Name, nodePoolName),

		ClusterID:       c.ID,
		ClusterSecretID: c.ConfigSecretID.String(),
		ClusterName:     c.Name,
		NodePoolName:    nodePoolName,
		OrganizationID:  c.OrganizationID,

		NodeImage: nodePoolUpdate.Image,
		Version:   nodePoolUpdate.Version,

		Options: pke.NodePoolUpdateOptions{
			MaxSurge:       nodePoolUpdate.Options.MaxSurge,
			MaxBatchSize:   nodePoolUpdate.Options.MaxBatchSize,
			MaxUnavailable: nodePoolUpdate.Options.MaxUnavailable,
			Drain: pke.NodePoolUpdateDrainOptions{
				Timeout:     nodePoolUpdate.Options.Drain.Timeout,
				FailOnError: nodePoolUpdate.Options.Drain.FailOnError,
				PodSelector: nodePoolUpdate.Options.Drain.PodSelector,
			},
		},
		ClusterTags: c.Tags,
	}

	e, err := n.workflowClient.StartWorkflow(ctx, workflowOptions, pkeawsworkflow.UpdateNodePoolWorkflowName, input)
	if err != nil {
		return "", errors.WrapWithDetails(err, "failed to start workflow", "workflow", pkeawsworkflow.UpdateNodePoolWorkflowName)
	}

	return e.ID, nil
}
