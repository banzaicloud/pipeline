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
)

const DeleteClusterFromStoreActivityName = "eks-delete-cluster-from-store"

type DeleteClusterFromStoreActivity struct {
	manager Clusters
}

func NewDeleteClusterFromStoreActivity(manager Clusters) DeleteClusterFromStoreActivity {
	return DeleteClusterFromStoreActivity{
		manager: manager,
	}
}

type DeleteClusterFromStoreActivityInput struct {
	ClusterID uint
}

func (a DeleteClusterFromStoreActivity) Execute(ctx context.Context, input DeleteClusterFromStoreActivityInput) error {
	cluster, err := a.manager.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}
	return cluster.DeleteFromDatabase()
}
