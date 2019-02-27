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
	"fmt"

	"github.com/banzaicloud/pipeline/internal/cluster/auth"
	"github.com/goph/emperror"
)

const CreateDexClientActivityName = "pke-create-dex-client-activity"

type CreateDexClientActivity struct {
	clusters           Clusters
	clusterAuthService auth.ClusterAuthService
}

func NewCreateDexClientActivity(clusters Clusters, clusterAuthService auth.ClusterAuthService) *CreateDexClientActivity {
	return &CreateDexClientActivity{
		clusters:           clusters,
		clusterAuthService: clusterAuthService,
	}
}

type CreateDexClientActivityInput struct {
	ClusterID uint
}

func (a *CreateDexClientActivity) Execute(ctx context.Context, input CreateDexClientActivityInput) error {
	cluster, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	err = a.clusterAuthService.RegisterCluster(ctx, cluster.GetName(), cluster.GetID(), cluster.GetUID())
	if err != nil {
		return emperror.Wrap(err, fmt.Sprintf("can't create dex client for cluster %+v", cluster))
	}

	return nil
}
