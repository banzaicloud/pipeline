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
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"go.uber.org/cadence/workflow"
)

const DeleteClusterFromStoreActivityName = "pke-azure-delete-cluster-from-store"

type DeleteClusterFromStoreActivity struct {
	store pke.AzurePKEClusterStore
}

func MakeDeleteClusterFromAtoreActivity(store pke.AzurePKEClusterStore) DeleteClusterFromStoreActivity {
	return DeleteClusterFromStoreActivity{
		store: store,
	}
}

type DeleteClusterFromStoreActivityInput struct {
	ClusterID uint
}

func (a DeleteClusterFromStoreActivity) Execute(ctx workflow.Context, input DeleteClusterFromStoreActivityInput) error {
	return a.store.Delete(input.ClusterID)
}
