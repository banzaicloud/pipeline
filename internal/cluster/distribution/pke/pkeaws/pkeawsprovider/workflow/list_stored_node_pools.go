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
	"fmt"

	"emperror.dev/errors"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
)

const ListStoredNodePoolsActivityName = "pke-aws-list-stored-node-pools"

// ListStoredNodePoolsActivity collects the necessary component dependencies
// for executing a stored node pool retrieval operation.
type ListStoredNodePoolsActivity struct {
	nodePoolStore pke.NodePoolStore
}

// ListStoredNodePoolsActivityInput encapsulates the dynamic parameters of the
// stored node pool retrieval operation.
type ListStoredNodePoolsActivityInput struct {
	ClusterID                   uint
	ClusterName                 string
	OptionalListedNodePoolNames []string
	OrganizationID              uint
}

type ListStoredNodePoolsActivityOutput struct {
	NodePools map[string]pke.ExistingNodePool
}

// NewListStoredNodePoolsActivity instantiates an activity object for deleting
// stored node pools.
func NewListStoredNodePoolsActivity(nodePoolStore pke.NodePoolStore) *ListStoredNodePoolsActivity {
	return &ListStoredNodePoolsActivity{
		nodePoolStore: nodePoolStore,
	}
}

// Execute executes a stored node pool deletion operation with the specified
// input parameters.
func (a *ListStoredNodePoolsActivity) Execute(
	ctx context.Context, input ListStoredNodePoolsActivityInput,
) (output *ListStoredNodePoolsActivityOutput, err error) {
	nodePools, err := a.nodePoolStore.ListNodePools(ctx, input.OrganizationID, input.ClusterID, input.ClusterName)
	if err != nil {
		return nil, err
	}

	output = &ListStoredNodePoolsActivityOutput{
		NodePools: make(map[string]pke.ExistingNodePool, len(nodePools)),
	}

	if len(input.OptionalListedNodePoolNames) == 0 {
		for nodePoolName := range nodePools {
			output.NodePools[nodePoolName] = nodePools[nodePoolName]
		}
	} else {
		var errs []error
		var isExisting bool
		for _, nodePoolName := range input.OptionalListedNodePoolNames {
			output.NodePools[nodePoolName], isExisting = nodePools[nodePoolName]
			if !isExisting {
				errs = append(errs, errors.NewWithDetails(
					fmt.Sprintf("node pool %s not found", nodePoolName), "nodePools", nodePools,
				))
			}
		}

		if len(errs) != 0 {
			return nil, errors.Combine(errs...)
		}
	}

	return output, nil
}

// Register registers the stored node pool deletion activity.
func (a ListStoredNodePoolsActivity) Register() {
	activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: ListStoredNodePoolsActivityName})
}
