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

package cluster

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/pkg/providers"
)

//go:generate mockery -name NodePoolStore -inpkg -testonly
//go:generate mockery -name NodePoolManager -inpkg -testonly

func TestNodePoolService_DeleteNodePool(t *testing.T) {
	t.Run("cluster_not_found", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)
		{
			err := NotFoundError{ID: 1}
			clusterStore.On("GetCluster", ctx, uint(1)).Return(Cluster{}, err)
		}

		nodePoolStore := new(MockNodePoolStore)
		nodePoolManager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolManager)

		_, err := nodePoolService.DeleteNodePool(ctx, 1, "pool0")
		require.Error(t, err)

		assert.True(t, errors.Is(err, NotFoundError{ID: 1}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolManager.AssertExpectations(t)
	})

	t.Run("distribution_not_supported", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         "something",
			Distribution:  "xks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		nodePoolStore := new(MockNodePoolStore)
		nodePoolManager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolManager)

		_, err := nodePoolService.DeleteNodePool(ctx, 1, "pool0")
		require.Error(t, err)

		assert.True(t, errors.As(err, &NotSupportedDistributionError{}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolManager.AssertExpectations(t)
	})

	t.Run("node_pool_does_not_exist", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         providers.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(false, nil)

		nodePoolManager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolManager)

		deleted, err := nodePoolService.DeleteNodePool(ctx, 1, nodePoolName)
		require.NoError(t, err)

		assert.True(t, deleted)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolManager.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         providers.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)
		clusterStore.On("SetStatus", ctx, cluster.ID, Updating, "deleting node pool").Return(nil)

		const nodePoolName = "pool0"

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(true, nil)

		nodePoolManager := new(MockNodePoolManager)
		nodePoolManager.On("DeleteNodePool", ctx, cluster.ID, nodePoolName).Return(nil)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolManager)

		deleted, err := nodePoolService.DeleteNodePool(ctx, 1, nodePoolName)
		require.NoError(t, err)

		assert.False(t, deleted)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolManager.AssertExpectations(t)
	})
}
