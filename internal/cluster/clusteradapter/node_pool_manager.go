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

package clusteradapter

import (
	"context"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/cluster/clusterworkflow"
)

// NodePoolManager manages node pool asynchronously via Cadence workflows.
type NodePoolManager struct {
	workflowClient client.Client
}

// NewNodePoolManager returns a new NodePoolManager.
func NewNodePoolManager(workflowClient client.Client) NodePoolManager {
	return NodePoolManager{
		workflowClient: workflowClient,
	}
}

func (n NodePoolManager) DeleteNodePool(ctx context.Context, clusterID uint, name string) error {
	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 30 * 24 * 60 * time.Minute,
	}

	input := clusterworkflow.DeleteNodePoolWorkflowInput{
		ClusterID:    clusterID,
		NodePoolName: name,
	}

	_, err := n.workflowClient.StartWorkflow(ctx, workflowOptions, clusterworkflow.DeleteNodePoolWorkflowName, input)
	if err != nil {
		return errors.WrapWithDetails(err, "failed to start workflow", "workflow", clusterworkflow.DeleteNodePoolWorkflowName)
	}

	return nil
}
