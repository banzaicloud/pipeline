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

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/providers"
)

const CreateNodePoolActivityName = "create-node-pool"

type CreateNodePoolActivity struct {
	clusters           cluster.Store
	eksNodePoolCreator NodePoolCreator
}

// NodePoolCreator creates a new node pool.
// This is a temporary interface to decouple some EKS specific logic from the clusterworkflow package.
// Once we get rid of the common node pool create workflow, this will go away as well.
type NodePoolCreator interface {
	CreateNodePool(
		ctx context.Context,
		userID uint,
		c cluster.Cluster,
		rawNodePool cluster.NewRawNodePool,
	) error
}

// NewCreateNodePoolActivity returns a new CreateNodePoolActivity.
func NewCreateNodePoolActivity(
	clusters cluster.Store,
	eksNodePoolCreator NodePoolCreator,
) CreateNodePoolActivity {
	return CreateNodePoolActivity{
		clusters:           clusters,
		eksNodePoolCreator: eksNodePoolCreator,
	}
}

type CreateNodePoolActivityInput struct {
	ClusterID   uint
	UserID      uint
	RawNodePool cluster.NewRawNodePool
}

func (a CreateNodePoolActivity) Execute(ctx context.Context, input CreateNodePoolActivityInput) error {
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return cadence.WrapClientError(err)
	}

	switch {
	case c.Cloud == providers.Amazon && c.Distribution == "eks":
		return a.eksNodePoolCreator.CreateNodePool(ctx, input.UserID, c, input.RawNodePool)
	default:
		return cadence.WrapClientError(errors.WithStack(cluster.NotSupportedDistributionError{
			ID:           c.ID,
			Cloud:        c.Cloud,
			Distribution: c.Distribution,

			Message: "the node pool API does not support this distribution yet",
		}))
	}
}
