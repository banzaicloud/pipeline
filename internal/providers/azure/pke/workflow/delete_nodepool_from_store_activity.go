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
	"context"

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/goph/emperror"
)

const DeleteNodePoolFromStoreActivityName = "pke-azure-delete-node-pool-from-store"

type DeleteNodePoolFromStoreActivity struct {
	store pke.AzurePKEClusterStore
}

func MakeDeleteNodePoolFromStoreActivity(store pke.AzurePKEClusterStore) DeleteNodePoolFromStoreActivity {
	return DeleteNodePoolFromStoreActivity{
		store: store,
	}
}

type DeleteNodePoolFromStoreActivityInput struct {
	ClusterID     uint
	NodePoolNames []string
}

func (a DeleteNodePoolFromStoreActivity) Execute(ctx context.Context, input DeleteNodePoolFromStoreActivityInput) error {
	for _, name := range input.NodePoolNames {
		err := a.store.DeleteNodePool(input.ClusterID, name)
		if err != nil {
			return emperror.WrapWith(err, "failed to delete nodepool from store", "nodepool", name, "cluster", input.ClusterID)
		}
	}
	return nil
}
