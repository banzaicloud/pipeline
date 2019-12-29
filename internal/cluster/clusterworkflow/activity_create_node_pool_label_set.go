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

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution"
	"github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/kubernetes/custom/npls"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/src/cluster/nodelabels"
)

const CreateNodePoolLabelSetActivityName = "create-node-pool-label-set"

type CreateNodePoolLabelSetActivity struct {
	clusters             cluster.Store
	dynamicClientFactory DynamicClientFactory
	namespace            string
}

// NewCreateNodePoolLabelSetActivity returns a new CreateNodePoolLabelSetActivity.
func NewCreateNodePoolLabelSetActivity(
	clusters cluster.Store,
	dynamicClientFactory DynamicClientFactory,
	namespace string,
) CreateNodePoolLabelSetActivity {
	return CreateNodePoolLabelSetActivity{
		clusters:             clusters,
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

	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return cadence.WrapClientError(err)
	}

	var name string
	var labels map[string]string

	switch {
	case c.Cloud == providers.Amazon && c.Distribution == "eks":
		var nodePool distribution.NewEKSNodePool

		err := mapstructure.Decode(input.RawNodePool, &nodePool)
		if err != nil {
			return errors.Wrap(err, "failed to decode node pool")
		}

		name = nodePool.Name

		labelNodePoolInfo := nodelabels.NodePoolInfo{
			Name:         nodePool.Name,
			SpotPrice:    nodePool.SpotPrice,
			InstanceType: nodePool.InstanceType,
			Labels:       nodePool.Labels,
		}

		labels = nodelabels.GetDesiredLabelsForNodePool(
			labelNodePoolInfo,
			false,
			c.Cloud,
			c.Distribution,
			c.Location,
		)
	}

	manager := npls.NewManager(client, a.namespace)

	err = manager.SyncOne(name, labels)
	if err != nil {
		return err
	}

	return nil
}
