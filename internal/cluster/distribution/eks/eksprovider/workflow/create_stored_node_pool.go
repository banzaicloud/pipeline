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
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

// CreateStoredNodePoolActivityName is the name of the stored node pool creation
// activity.
const CreateStoredNodePoolActivityName = "eks-create-stored-node-pool"

// CreateStoredNodePoolActivity collects the necessary component dependencies
// for executing a stored node pool deletion operation.
type CreateStoredNodePoolActivity struct {
	nodePoolStore eks.NodePoolStore
}

// CreateStoredNodePoolActivityInput encapsulates the dynamic parameters of the
// stored node pool deletion operation.
type CreateStoredNodePoolActivityInput struct {
	ClusterID      uint
	ClusterName    string
	NodePool       eks.NewNodePool
	OrganizationID uint
	UserID         uint
}

// NewCreateStoredNodePoolActivity instantiates an activity object for deleting
// stored node pools.
func NewCreateStoredNodePoolActivity(nodePoolStore eks.NodePoolStore) *CreateStoredNodePoolActivity {
	return &CreateStoredNodePoolActivity{
		nodePoolStore: nodePoolStore,
	}
}

// Execute executes a stored node pool deletion operation with the specified
// input parameters.
func (a *CreateStoredNodePoolActivity) Execute(ctx context.Context, input CreateStoredNodePoolActivityInput) error {
	return a.nodePoolStore.CreateNodePool(
		ctx, input.OrganizationID, input.ClusterID, input.ClusterName, input.UserID, input.NodePool,
	)
}

// Register registers the stored node pool deletion activity.
func (a CreateStoredNodePoolActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: CreateStoredNodePoolActivityName})
}

// createStoredNodePool creates a stored node pool and returns an error if one
// occurs.
//
// This is a convenience wrapper around the corresponding activity.
func createStoredNodePool(
	ctx workflow.Context,
	organizationID uint,
	clusterID uint,
	clusterName string,
	userID uint,
	nodePool eks.NewNodePool,
) error {
	return createStoredNodePoolAsync(ctx, organizationID, clusterID, clusterName, userID, nodePool).Get(ctx, nil)
}

// createStoredNodePoolAsync returns a future object for creating a stored node
// pool with the specified arguments.
//
// This is a convenience wrapper around the corresponding activity.
func createStoredNodePoolAsync(
	ctx workflow.Context,
	organizationID uint,
	clusterID uint,
	clusterName string,
	userID uint,
	nodePool eks.NewNodePool,
) workflow.Future {
	return workflow.ExecuteActivity(ctx, CreateStoredNodePoolActivityName, CreateStoredNodePoolActivityInput{
		ClusterID:      clusterID,
		ClusterName:    clusterName,
		NodePool:       nodePool,
		OrganizationID: organizationID,
		UserID:         userID,
	})
}
