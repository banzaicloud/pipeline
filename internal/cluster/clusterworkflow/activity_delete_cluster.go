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

package clusterworkflow

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

const DeleteClusterActivityName = "delete-cluster"

type DeleteClusterActivity struct {
	clusterDeleter cluster.Deleter
}

func MakeDeleteClusterActivity(clusterDeleter cluster.Deleter) DeleteClusterActivity {
	return DeleteClusterActivity{
		clusterDeleter: clusterDeleter,
	}
}

type DeleteClusterActivityInput struct {
	ClusterID uint
	Force     bool
}

func (a DeleteClusterActivity) Execute(ctx context.Context, input DeleteClusterActivityInput) error {
	return a.clusterDeleter.DeleteCluster(ctx, input.ClusterID, cluster.DeleteClusterOptions{Force: input.Force})
}
