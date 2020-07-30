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
	"github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/kubernetes/custom/npls"
)

const CreateNodePoolLabelSetActivityName = "create-node-pool-label-set"

type CreateNodePoolLabelSetActivity struct {
	dynamicClientFactory DynamicClientFactory
	namespace            string
}

// NewCreateNodePoolLabelSetActivity returns a new CreateNodePoolLabelSetActivity.
func NewCreateNodePoolLabelSetActivity(
	dynamicClientFactory DynamicClientFactory,
	namespace string,
) CreateNodePoolLabelSetActivity {
	return CreateNodePoolLabelSetActivity{
		dynamicClientFactory: dynamicClientFactory,
		namespace:            namespace,
	}
}

type CreateNodePoolLabelSetActivityInput struct {
	ClusterID   uint
	RawNodePool cluster.NewRawNodePool
}

func (a CreateNodePoolLabelSetActivity) Execute(ctx context.Context, input CreateNodePoolLabelSetActivityInput) error {
	client, err := a.dynamicClientFactory.FromClusterID(ctx, input.ClusterID)
	if err != nil {
		return cadence.WrapClientError(err)
	}

	manager := npls.NewManager(client, a.namespace)

	err = manager.SyncOne(ctx, input.RawNodePool.GetName(), input.RawNodePool.GetLabels())
	if err != nil {
		return err
	}

	return nil
}
