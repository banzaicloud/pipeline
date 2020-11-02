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
	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/clusterworkflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	pkgcadence "github.com/banzaicloud/pipeline/pkg/cadence"
	sdkcadence "github.com/banzaicloud/pipeline/pkg/sdk/cadence"
)

// createNodePoolLabelSet creates the corresponding Kubernetes node pool label
// set for the specified node pool.
//
// This is a convenience wrapper around the corresponding activity.
func createNodePoolLabelSetFromEKSNodePool(ctx workflow.Context, clusterID uint, nodePool eks.NewNodePool) error {
	return createNodePoolLabelSetFromEKSNodePoolAsync(ctx, clusterID, nodePool).Get(ctx, nil)
}

// createNodePoolLabelSetAsync returns a future object for creating the
// corresponding Kubernetes node pool label set for the specified node pool.
//
// This is a convenience wrapper around the corresponding activity.
func createNodePoolLabelSetFromEKSNodePoolAsync(
	ctx workflow.Context,
	clusterID uint,
	nodePool eks.NewNodePool,
) workflow.Future {
	rawNodePool := cluster.NewRawNodePool(map[string]interface{}{})
	err := mapstructure.Decode(nodePool, &rawNodePool)
	if err != nil {
		return sdkcadence.NewReadyFuture(ctx, nil, pkgcadence.NewClientError(
			errors.WrapIfWithDetails(
				err,
				"transforming node pool to raw node pool failed",
				"nodePool", nodePool,
			),
		))
	}

	return workflow.ExecuteActivity(ctx, clusterworkflow.CreateNodePoolLabelSetActivityName, clusterworkflow.CreateNodePoolLabelSetActivityInput{
		ClusterID:   clusterID,
		RawNodePool: rawNodePool,
	})
}
