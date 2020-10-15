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

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/clusterworkflow"
)

type nodePoolManager struct {
	workflowClient client.Client
	getUserID      func(ctx context.Context) uint
}

// NewNodePoolManager returns a new cluster.NodePoolManager
// that manages node pools asynchronously via Cadence workflows.
func NewNodePoolManager(workflowClient client.Client, getUserID func(ctx context.Context) uint) cluster.NodePoolManager {
	return nodePoolManager{
		workflowClient: workflowClient,
		getUserID:      getUserID,
	}
}

func (n nodePoolManager) CreateNodePool(
	ctx context.Context,
	clusterID uint,
	rawNodePool cluster.NewRawNodePool,
) error {
	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 30 * 24 * 60 * time.Minute,
	}

	input := clusterworkflow.CreateNodePoolWorkflowInput{
		ClusterID:   clusterID,
		UserID:      n.getUserID(ctx),
		RawNodePool: rawNodePool,
	}

	_, err := n.workflowClient.StartWorkflow(ctx, workflowOptions, clusterworkflow.CreateNodePoolWorkflowName, input)
	if err != nil {
		return errors.WrapWithDetails(err, "failed to start workflow", "workflow", clusterworkflow.CreateNodePoolWorkflowName)
	}

	return nil
}
