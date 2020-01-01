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

//go:generate mga gen mockery --name NodePoolStore --inpkg --testonly
//go:generate mga gen mockery --name NodePoolValidator --inpkg --testonly
//go:generate mga gen mockery --name NodePoolProcessor --inpkg --testonly
//go:generate mga gen mockery --name NodePoolManager --inpkg --testonly

func TestNodePoolService_CreateNodePool(t *testing.T) {
	t.Run("cluster_not_found", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)
		{
			err := NotFoundError{ID: 1}
			clusterStore.On("GetCluster", ctx, uint(1)).Return(Cluster{}, err)
		}

		nodePoolStore := new(MockNodePoolStore)
		nodePoolValidator := new(MockNodePoolValidator)
		nodePoolProcessor := new(MockNodePoolProcessor)
		nodePoolManager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolValidator, nodePoolProcessor, nodePoolManager)

		rawNewNodePool := NewRawNodePool{
			"name": "pool0",
		}

		err := nodePoolService.CreateNodePool(ctx, 1, rawNewNodePool)
		require.Error(t, err)

		assert.True(t, errors.Is(err, NotFoundError{ID: 1}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolValidator.AssertExpectations(t)
		nodePoolProcessor.AssertExpectations(t)
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
		nodePoolValidator := new(MockNodePoolValidator)
		nodePoolProcessor := new(MockNodePoolProcessor)
		nodePoolManager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolValidator, nodePoolProcessor, nodePoolManager)

		rawNewNodePool := NewRawNodePool{
			"name": "pool0",
		}

		err := nodePoolService.CreateNodePool(ctx, 1, rawNewNodePool)
		require.Error(t, err)

		assert.True(t, errors.As(err, &NotSupportedDistributionError{}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolValidator.AssertExpectations(t)
		nodePoolProcessor.AssertExpectations(t)
		nodePoolManager.AssertExpectations(t)
	})

	t.Run("invalid_node_pool", func(t *testing.T) {
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

		rawNewNodePool := NewRawNodePool{
			"name": nodePoolName,
		}

		nodePoolStore := new(MockNodePoolStore)

		validationError := errors.New("invalid node pool")

		nodePoolValidator := new(MockNodePoolValidator)
		nodePoolValidator.On("ValidateNew", ctx, cluster, rawNewNodePool).Return(validationError)

		nodePoolProcessor := new(MockNodePoolProcessor)
		nodePoolManager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolValidator, nodePoolProcessor, nodePoolManager)

		err := nodePoolService.CreateNodePool(ctx, 1, rawNewNodePool)
		require.Error(t, err)

		assert.Equal(t, validationError, err)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolValidator.AssertExpectations(t)
		nodePoolProcessor.AssertExpectations(t)
		nodePoolManager.AssertExpectations(t)
	})

	t.Run("node_pool_already_exist", func(t *testing.T) {
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

		rawNewNodePool := NewRawNodePool{
			"name": nodePoolName,
		}

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(true, nil)

		nodePoolValidator := new(MockNodePoolValidator)
		nodePoolValidator.On("ValidateNew", ctx, cluster, rawNewNodePool).Return(nil)

		nodePoolProcessor := new(MockNodePoolProcessor)
		nodePoolManager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolValidator, nodePoolProcessor, nodePoolManager)

		err := nodePoolService.CreateNodePool(ctx, 1, rawNewNodePool)
		require.Error(t, err)

		assert.True(t, errors.As(err, &NodePoolAlreadyExistsError{}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolValidator.AssertExpectations(t)
		nodePoolProcessor.AssertExpectations(t)
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
		clusterStore.On("SetStatus", ctx, cluster.ID, Updating, "creating node pool").Return(nil)

		const nodePoolName = "pool0"

		rawNewNodePool := NewRawNodePool{
			"name": nodePoolName,
		}

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(false, nil)

		nodePoolValidator := new(MockNodePoolValidator)
		nodePoolValidator.On("ValidateNew", ctx, cluster, rawNewNodePool).Return(nil)

		nodePoolProcessor := new(MockNodePoolProcessor)
		nodePoolProcessor.On("ProcessNew", ctx, cluster, rawNewNodePool).Return(rawNewNodePool, nil)

		nodePoolManager := new(MockNodePoolManager)
		nodePoolManager.On("CreateNodePool", ctx, cluster.ID, rawNewNodePool).Return(nil)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolValidator, nodePoolProcessor, nodePoolManager)

		err := nodePoolService.CreateNodePool(ctx, 1, rawNewNodePool)
		require.NoError(t, err)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolValidator.AssertExpectations(t)
		nodePoolProcessor.AssertExpectations(t)
		nodePoolManager.AssertExpectations(t)
	})
}

func TestNodePoolService_DeleteNodePool(t *testing.T) {
	t.Run("cluster_not_found", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)
		{
			err := NotFoundError{ID: 1}
			clusterStore.On("GetCluster", ctx, uint(1)).Return(Cluster{}, err)
		}

		nodePoolStore := new(MockNodePoolStore)
		nodePoolValidator := new(MockNodePoolValidator)
		nodePoolProcessor := new(MockNodePoolProcessor)
		nodePoolManager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolValidator, nodePoolProcessor, nodePoolManager)

		_, err := nodePoolService.DeleteNodePool(ctx, 1, "pool0")
		require.Error(t, err)

		assert.True(t, errors.Is(err, NotFoundError{ID: 1}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolValidator.AssertExpectations(t)
		nodePoolProcessor.AssertExpectations(t)
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
		nodePoolValidator := new(MockNodePoolValidator)
		nodePoolProcessor := new(MockNodePoolProcessor)
		nodePoolManager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolValidator, nodePoolProcessor, nodePoolManager)

		_, err := nodePoolService.DeleteNodePool(ctx, 1, "pool0")
		require.Error(t, err)

		assert.True(t, errors.As(err, &NotSupportedDistributionError{}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolValidator.AssertExpectations(t)
		nodePoolProcessor.AssertExpectations(t)
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

		nodePoolValidator := new(MockNodePoolValidator)
		nodePoolProcessor := new(MockNodePoolProcessor)
		nodePoolManager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolValidator, nodePoolProcessor, nodePoolManager)

		deleted, err := nodePoolService.DeleteNodePool(ctx, 1, nodePoolName)
		require.NoError(t, err)

		assert.True(t, deleted)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolValidator.AssertExpectations(t)
		nodePoolProcessor.AssertExpectations(t)
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

		nodePoolValidator := new(MockNodePoolValidator)
		nodePoolProcessor := new(MockNodePoolProcessor)

		nodePoolManager := new(MockNodePoolManager)
		nodePoolManager.On("DeleteNodePool", ctx, cluster.ID, nodePoolName).Return(nil)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, nodePoolValidator, nodePoolProcessor, nodePoolManager)

		deleted, err := nodePoolService.DeleteNodePool(ctx, 1, nodePoolName)
		require.NoError(t, err)

		assert.False(t, deleted)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		nodePoolValidator.AssertExpectations(t)
		nodePoolProcessor.AssertExpectations(t)
		nodePoolManager.AssertExpectations(t)
	})
}
