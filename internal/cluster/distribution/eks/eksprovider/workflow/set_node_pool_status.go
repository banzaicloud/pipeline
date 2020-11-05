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

package workflow

import (
	"context"

	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

// SetNodePoolStatusActivityName is the name of the activity which sets a node
// pool's status.
const SetNodePoolStatusActivityName = "eks-set-node-pool-status"

// SetNodePoolStatusActivity collects the static high level component objects
// required for setting a node pool\s status.
type SetNodePoolStatusActivity struct {
	nodePoolStore eks.NodePoolStore
}

// NewSetNodePoolStatusActivity instantiates a node pool status setting
// activity.
func NewSetNodePoolStatusActivity(nodePoolStore eks.NodePoolStore) (activity *SetNodePoolStatusActivity) {
	return &SetNodePoolStatusActivity{
		nodePoolStore: nodePoolStore,
	}
}

// SetNodePoolStatusActivityInput collects the required parameters for setting a
// node pool\s status.
type SetNodePoolStatusActivityInput struct {
	ClusterID             uint
	ClusterName           string
	NodePoolName          string
	NodePoolStatus        eks.NodePoolStatus
	NodePoolStatusMessage string
	OrganizationID        uint
}

// Execute executes the activity.
func (a SetNodePoolStatusActivity) Execute(ctx context.Context, input SetNodePoolStatusActivityInput) (err error) {
	return a.nodePoolStore.UpdateNodePoolStatus(
		ctx,
		input.OrganizationID,
		input.ClusterID,
		input.ClusterName,
		input.NodePoolName,
		input.NodePoolStatus,
		input.NodePoolStatusMessage,
	)
}

// Register registers the activity.
func (a SetNodePoolStatusActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: SetNodePoolStatusActivityName})
}

// setNodePoolErrorStatus sets a node pool's status to error using the
// corresponding activity and the specified arguments.
func setNodePoolErrorStatus(
	ctx workflow.Context, organizationID, clusterID uint, clusterName, nodePoolName string, statusError error,
) (err error) {
	statusMessage := ""
	if statusError != nil {
		statusMessage = pkgCadence.UnwrapError(statusError).Error()
	}

	return setNodePoolStatus(
		ctx,
		organizationID,
		clusterID,
		clusterName,
		nodePoolName,
		eks.NodePoolStatusError,
		statusMessage,
	)
}

// setNodePoolStatus sets a node pool's status using the specified arguments.
//
// This is a convenience wrapper around the corresponding activity.
func setNodePoolStatus(
	ctx workflow.Context,
	organizationID uint,
	clusterID uint,
	clusterName string,
	nodePoolName string,
	nodePoolStatus eks.NodePoolStatus,
	nodePoolStatusMessage string,
) error {
	return setNodePoolStatusAsync(
		ctx,
		organizationID,
		clusterID,
		clusterName,
		nodePoolName,
		nodePoolStatus,
		nodePoolStatusMessage,
	).Get(ctx, nil)
}

// setNodePoolStatus returns a future for setting a node pool's status using the
// specified arguments.
//
// This is a convenience wrapper around the corresponding activity.
func setNodePoolStatusAsync(
	ctx workflow.Context,
	organizationID uint,
	clusterID uint,
	clusterName string,
	nodePoolName string,
	nodePoolStatus eks.NodePoolStatus,
	nodePoolStatusMessage string,
) workflow.Future {
	return workflow.ExecuteActivity(ctx, SetNodePoolStatusActivityName, SetNodePoolStatusActivityInput{
		ClusterID:             clusterID,
		ClusterName:           clusterName,
		NodePoolName:          nodePoolName,
		NodePoolStatus:        nodePoolStatus,
		NodePoolStatusMessage: nodePoolStatusMessage,
		OrganizationID:        organizationID,
	})
}
