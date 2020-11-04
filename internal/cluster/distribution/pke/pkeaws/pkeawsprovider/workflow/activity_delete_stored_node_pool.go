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

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
)

const DeleteStoredNodePoolActivityName = "pke-aws-delete-stored-node-pool"

// DeleteStoredNodePoolActivity collects the necessary component dependencies
// for executing a stored node pool deletion operation.
type DeleteStoredNodePoolActivity struct {
	nodePoolStore pke.NodePoolStore
}

// DeleteStoredNodePoolActivityInput encapsulates the dynamic parameters of the
// stored node pool deletion operation.
type DeleteStoredNodePoolActivityInput struct {
	ClusterID      uint
	ClusterName    string
	NodePoolName   string
	OrganizationID uint
}

// NewDeleteStoredNodePoolActivity instantiates an activity object for deleting
// stored node pools.
func NewDeleteStoredNodePoolActivity(nodePoolStore pke.NodePoolStore) *DeleteStoredNodePoolActivity {
	return &DeleteStoredNodePoolActivity{
		nodePoolStore: nodePoolStore,
	}
}

// Execute executes a stored node pool deletion operation with the specified
// input parameters.
func (a *DeleteStoredNodePoolActivity) Execute(ctx context.Context, input DeleteStoredNodePoolActivityInput) error {
	return a.nodePoolStore.DeleteNodePool(
		ctx, input.OrganizationID, input.ClusterID, input.ClusterName, input.NodePoolName,
	)
}

// Register registers the stored node pool deletion activity.
func (a DeleteStoredNodePoolActivity) Register() {
	activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: DeleteStoredNodePoolActivityName})
}
