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

	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
)

const DeleteNodePoolFromStoreActivityName = "pke-vsphere-delete-node-pool-from-store"

type DeleteNodePoolFromStoreActivity struct {
	store pke.ClusterStore
}

func MakeDeleteNodePoolFromStoreActivity(store pke.ClusterStore) DeleteNodePoolFromStoreActivity {
	return DeleteNodePoolFromStoreActivity{
		store: store,
	}
}

type DeleteNodePoolFromStoreActivityInput struct {
	ClusterID    uint
	NodePoolName string
}

func (a DeleteNodePoolFromStoreActivity) Execute(ctx context.Context, input DeleteNodePoolFromStoreActivityInput) error {
	return a.store.DeleteNodePool(input.ClusterID, input.NodePoolName)
}
