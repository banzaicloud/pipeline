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

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
)

const SaveClusterVersionActivityName = "eks-save-cluster-version"

type SaveClusterVersionActivity struct {
	manager Clusters
}

func NewSaveClusterVersionActivity(manager Clusters) SaveClusterVersionActivity {
	return SaveClusterVersionActivity{
		manager: manager,
	}
}

type SaveClusterVersionActivityInput struct {
	ClusterID uint
	Version   string
}

func (a SaveClusterVersionActivity) Execute(ctx context.Context, input SaveClusterVersionActivityInput) error {
	cluster, err := a.manager.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	if eksCluster, ok := cluster.(interface {
		GetModel() *eksmodel.EKSClusterModel
	}); ok {
		modelCluster := eksCluster.GetModel()
		modelCluster.Version = input.Version
	}

	return cluster.Persist()
}
