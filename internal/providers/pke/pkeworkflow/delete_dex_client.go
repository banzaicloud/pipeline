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

package pkeworkflow

import (
	"context"

	"emperror.dev/emperror"

	"github.com/banzaicloud/pipeline/internal/cluster/auth"
)

const DeleteDexClientActivityName = "pke-delete-dex-client-activity"

type DeleteDexClientActivity struct {
	clusters           Clusters
	clusterAuthService auth.ClusterAuthService
}

func NewDeleteDexClientActivity(clusters Clusters, clusterAuthService auth.ClusterAuthService) *DeleteDexClientActivity {
	return &DeleteDexClientActivity{
		clusters:           clusters,
		clusterAuthService: clusterAuthService,
	}
}

type DeleteDexClientActivityInput struct {
	ClusterID uint
}

func (a *DeleteDexClientActivity) Execute(ctx context.Context, input DeleteDexClientActivityInput) error {
	cluster, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	err = a.clusterAuthService.UnRegisterCluster(ctx, cluster.GetUID())
	if err != nil {
		return emperror.Wrap(err, "can't delete dex client for cluster")
	}

	return nil
}
