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
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"go.uber.org/cadence/workflow"
)

const SetClusterStatusActivityName = "pke-azure-set-cluster-status"

type SetClusterStatusActivity struct {
	store pke.AzurePKEClusterStore
}

func MakeSetClusterStatusActivity(store pke.AzurePKEClusterStore) SetClusterStatusActivity {
	return SetClusterStatusActivity{
		store: store,
	}
}

type SetClusterStatusActivityInput struct {
	ClusterID     uint
	Status        string
	StatusMessage string
}

func (a SetClusterStatusActivity) Execute(ctx context.Context, input SetClusterStatusActivityInput) error {
	return a.store.SetStatus(input.ClusterID, input.Status, input.StatusMessage)
}

func setClusterErrorStatus(ctx workflow.Context, clusterID uint, err error) {
	workflow.ExecuteActivity(ctx, SetClusterStatusActivityName, SetClusterStatusActivityInput{
		ClusterID:     clusterID,
		Status:        pkgCluster.Error,
		StatusMessage: err.Error(),
	})
}
