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

package eksadapter

import (
	"context"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksworkflow"
)

type clusterManager struct {
	workflowClient client.Client
}

// NewClusterManager returns a new eks.ClusterManager
// that manages clusters asynchronously via Cadence workflows.
func NewClusterManager(workflowClient client.Client) eks.ClusterManager {
	return clusterManager{
		workflowClient: workflowClient,
	}
}

func (n clusterManager) UpdateCluster(
	ctx context.Context,
	c cluster.Cluster,
	clusterUpdate eks.ClusterUpdate,
) error {
	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 30 * 24 * 60 * time.Minute,
	}

	input := eksworkflow.UpdateClusterWorkflowInput{
		ProviderSecretID: c.SecretID.ResourceID,
		Region:           c.Location,

		ClusterID:      c.ID,
		ClusterName:    c.Name,
		OrganizationID: c.OrganizationID,

		Version: clusterUpdate.Version,
	}

	_, err := n.workflowClient.StartWorkflow(ctx, workflowOptions, eksworkflow.UpdateClusterWorkflowName, input)
	if err != nil {
		return errors.WrapWithDetails(err, "failed to start workflow", "workflow", eksworkflow.UpdateNodePoolWorkflowName)
	}

	return nil
}
