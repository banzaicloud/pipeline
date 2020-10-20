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
