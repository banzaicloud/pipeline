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

func TestNewRawNodePool(t *testing.T) {
	t.Run("GetName", func(t *testing.T) {
		np := NewRawNodePool{
			"name": "pool0",
		}

		assert.Equal(t, "pool0", np.GetName())
	})

	t.Run("GetInstanceType", func(t *testing.T) {
		np := NewRawNodePool{
			"instanceType": "t2-medium",
		}

		assert.Equal(t, "t2-medium", np.GetInstanceType())
	})

	t.Run("IsOnDemand", func(t *testing.T) {
		t.Run("ondemand", func(t *testing.T) {
			np := NewRawNodePool{}

			assert.True(t, np.IsOnDemand())
		})

		t.Run("spot", func(t *testing.T) {
			np := NewRawNodePool{
				"spotPrice": "0.1",
			}

			assert.False(t, np.IsOnDemand())
		})

		t.Run("preemptible", func(t *testing.T) {
			np := NewRawNodePool{
				"preemptible": true,
			}

			assert.False(t, np.IsOnDemand())
		})
	})

	t.Run("GetLabels", func(t *testing.T) {
		t.Run("string", func(t *testing.T) {
			np := NewRawNodePool{
				"labels": map[string]string{
					"key": "value",
				},
			}

			assert.Equal(t, map[string]string{"key": "value"}, np.GetLabels())
		})

		t.Run("interface", func(t *testing.T) {
			np := NewRawNodePool{
				"labels": map[string]interface{}{
					"key": "value",
				},
			}

			assert.Equal(t, map[string]string{"key": "value"}, np.GetLabels())
		})

		t.Run("empty", func(t *testing.T) {
			np := NewRawNodePool{}

			assert.Equal(t, map[string]string{}, np.GetLabels())
		})

		t.Run("wrong_type", func(t *testing.T) {
			np := NewRawNodePool{
				"labels": map[string]int{
					"key": 1,
				},
			}

			assert.Equal(t, map[string]string{}, np.GetLabels())
		})
	})
}

type nodePoolStub struct {
	name         string
	instanceType string
	onDemand     bool
	labels       map[string]string
}

func (n nodePoolStub) GetName() string {
	return n.name
}

func (n nodePoolStub) GetInstanceType() string {
	return n.instanceType
}

func (n nodePoolStub) IsOnDemand() bool {
	return n.onDemand
}

func (n nodePoolStub) GetLabels() map[string]string {
	return n.labels
}

func TestNodePoolService_CreateNodePool(t *testing.T) {
	t.Run("cluster_not_found", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)
		{
			err := NotFoundError{ClusterID: 1}
			clusterStore.On("GetCluster", ctx, uint(1)).Return(Cluster{}, err)
		}

		nodePoolStore := new(MockNodePoolStore)
		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		manager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, validator, processor, manager)

		rawNewNodePool := NewRawNodePool{
			"name": "pool0",
		}

		err := nodePoolService.CreateNodePool(ctx, 1, rawNewNodePool)
		require.Error(t, err)

		assert.True(t, errors.Is(err, NotFoundError{ClusterID: 1}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
		manager.AssertExpectations(t)
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
		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		manager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, validator, processor, manager)

		rawNewNodePool := NewRawNodePool{
			"name": "pool0",
		}

		err := nodePoolService.CreateNodePool(ctx, 1, rawNewNodePool)
		require.Error(t, err)

		assert.True(t, errors.As(err, &NotSupportedDistributionError{}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
		manager.AssertExpectations(t)
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

		validator := new(MockNodePoolValidator)
		validator.On("ValidateNew", ctx, cluster, rawNewNodePool).Return(validationError)

		processor := new(MockNodePoolProcessor)
		manager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, validator, processor, manager)

		err := nodePoolService.CreateNodePool(ctx, 1, rawNewNodePool)
		require.Error(t, err)

		assert.Equal(t, validationError, err)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
		manager.AssertExpectations(t)
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

		validator := new(MockNodePoolValidator)
		validator.On("ValidateNew", ctx, cluster, rawNewNodePool).Return(nil)

		processor := new(MockNodePoolProcessor)
		manager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, validator, processor, manager)

		err := nodePoolService.CreateNodePool(ctx, 1, rawNewNodePool)
		require.Error(t, err)

		assert.True(t, errors.As(err, &NodePoolAlreadyExistsError{}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
		manager.AssertExpectations(t)
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

		validator := new(MockNodePoolValidator)
		validator.On("ValidateNew", ctx, cluster, rawNewNodePool).Return(nil)

		processor := new(MockNodePoolProcessor)
		processor.On("ProcessNew", ctx, cluster, rawNewNodePool).Return(rawNewNodePool, nil)

		manager := new(MockNodePoolManager)
		manager.On("CreateNodePool", ctx, cluster.ID, rawNewNodePool).Return(nil)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, validator, processor, manager)

		err := nodePoolService.CreateNodePool(ctx, 1, rawNewNodePool)
		require.NoError(t, err)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
		manager.AssertExpectations(t)
	})
}

func TestNodePoolService_DeleteNodePool(t *testing.T) {
	t.Run("cluster_not_found", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)
		{
			err := NotFoundError{ClusterID: 1}
			clusterStore.On("GetCluster", ctx, uint(1)).Return(Cluster{}, err)
		}

		nodePoolStore := new(MockNodePoolStore)
		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		manager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, validator, processor, manager)

		_, err := nodePoolService.DeleteNodePool(ctx, 1, "pool0")
		require.Error(t, err)

		assert.True(t, errors.Is(err, NotFoundError{ClusterID: 1}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
		manager.AssertExpectations(t)
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
		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		manager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, validator, processor, manager)

		_, err := nodePoolService.DeleteNodePool(ctx, 1, "pool0")
		require.Error(t, err)

		assert.True(t, errors.As(err, &NotSupportedDistributionError{}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
		manager.AssertExpectations(t)
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

		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		manager := new(MockNodePoolManager)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, validator, processor, manager)

		deleted, err := nodePoolService.DeleteNodePool(ctx, 1, nodePoolName)
		require.NoError(t, err)

		assert.True(t, deleted)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
		manager.AssertExpectations(t)
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

		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)

		manager := new(MockNodePoolManager)
		manager.On("DeleteNodePool", ctx, cluster.ID, nodePoolName).Return(nil)

		nodePoolService := NewNodePoolService(clusterStore, nodePoolStore, validator, processor, manager)

		deleted, err := nodePoolService.DeleteNodePool(ctx, 1, nodePoolName)
		require.NoError(t, err)

		assert.False(t, deleted)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
		manager.AssertExpectations(t)
	})
}
