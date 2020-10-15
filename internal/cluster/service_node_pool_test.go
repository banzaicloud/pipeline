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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/pkg/cloud"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
)

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
		t.Run("OnDemand", func(t *testing.T) {
			np := NewRawNodePool{}

			assert.True(t, np.IsOnDemand())
		})

		t.Run("Spot", func(t *testing.T) {
			np := NewRawNodePool{
				"spotPrice": "0.1",
			}

			assert.False(t, np.IsOnDemand())
		})

		t.Run("Preemptible", func(t *testing.T) {
			np := NewRawNodePool{
				"preemptible": true,
			}

			assert.False(t, np.IsOnDemand())
		})
	})

	t.Run("GetLabels", func(t *testing.T) {
		t.Run("String", func(t *testing.T) {
			np := NewRawNodePool{
				"labels": map[string]string{
					"key": "value",
				},
			}

			assert.Equal(t, map[string]string{"key": "value"}, np.GetLabels())
		})

		t.Run("Interface", func(t *testing.T) {
			np := NewRawNodePool{
				"labels": map[string]interface{}{
					"key": "value",
				},
			}

			assert.Equal(t, map[string]string{"key": "value"}, np.GetLabels())
		})

		t.Run("Empty", func(t *testing.T) {
			np := NewRawNodePool{}

			assert.Equal(t, map[string]string{}, np.GetLabels())
		})

		t.Run("WrongType", func(t *testing.T) {
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
	t.Run("ClusterNotFound", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)
		{
			err := NotFoundError{ClusterID: 1}
			clusterStore.On("GetCluster", ctx, uint(1)).Return(Cluster{}, err)
		}

		nodePoolStore := new(MockNodePoolStore)
		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		rawNewNodePool := NewRawNodePool{
			"name": "pool0",
		}

		err := service.CreateNodePool(ctx, 1, rawNewNodePool)
		require.Error(t, err)

		assert.True(t, errors.Is(err, NotFoundError{ClusterID: 1}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("DistributionNotSupported", func(t *testing.T) {
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
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		rawNewNodePool := NewRawNodePool{
			"name": "pool0",
		}

		err := service.CreateNodePool(ctx, 1, rawNewNodePool)
		require.Error(t, err)

		assert.True(t, errors.As(err, &NotSupportedDistributionError{}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("InvalidNodePool", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
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
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		err := service.CreateNodePool(ctx, 1, rawNewNodePool)
		require.Error(t, err)

		assert.Equal(t, validationError, err)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("NodePoolExistsError", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		rawNewNodePool := NewRawNodePool{
			"name": nodePoolName,
		}

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(false, "", errors.New("node pool exists error"))

		validator := new(MockNodePoolValidator)
		validator.On("ValidateNew", ctx, cluster, rawNewNodePool).Return(nil)

		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		err := service.CreateNodePool(ctx, 1, rawNewNodePool)
		require.EqualError(t, err, "node pool exists error")

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("NodePoolAlreadyExists", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		rawNewNodePool := NewRawNodePool{
			"name": nodePoolName,
		}

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(true, nodePoolName, nil)

		validator := new(MockNodePoolValidator)
		validator.On("ValidateNew", ctx, cluster, rawNewNodePool).Return(nil)

		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		err := service.CreateNodePool(ctx, 1, rawNewNodePool)
		require.EqualError(t, err, NodePoolAlreadyExistsError{
			ClusterID: cluster.ID,
			NodePool:  nodePoolName,
		}.Error())

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("ProcessNewError", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		rawNewNodePool := NewRawNodePool{
			"name": nodePoolName,
		}

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(false, nodePoolName, nil)

		validator := new(MockNodePoolValidator)
		validator.On("ValidateNew", ctx, cluster, rawNewNodePool).Return(nil)

		processor := new(MockNodePoolProcessor)
		processor.On("ProcessNew", ctx, cluster, rawNewNodePool).Return(nil, errors.New("process new error"))

		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		err := service.CreateNodePool(ctx, 1, rawNewNodePool)
		require.EqualError(t, err, "process new error")

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("NotSupportedDistributionService", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		rawNewNodePool := NewRawNodePool{
			"name": nodePoolName,
		}

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(false, nodePoolName, nil)

		validator := new(MockNodePoolValidator)
		validator.On("ValidateNew", ctx, cluster, rawNewNodePool).Return(nil)

		processor := new(MockNodePoolProcessor)
		processor.On("ProcessNew", ctx, cluster, rawNewNodePool).Return(rawNewNodePool, nil)

		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		err := service.CreateNodePool(ctx, 1, rawNewNodePool)
		require.EqualError(t, err, NotSupportedDistributionError{
			ID:           cluster.ID,
			Cloud:        cluster.Cloud,
			Distribution: cluster.Distribution,

			Message: "not supported distribution",
		}.Error())

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("CreateNodePoolError", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		rawNewNodePool := NewRawNodePool{
			"name": nodePoolName,
		}

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(false, "", nil)

		distributionService := new(MockService)
		distributionService.On("CreateNodePool", ctx, cluster.ID, rawNewNodePool).
			Return(errors.New("create node pool error"))

		validator := new(MockNodePoolValidator)
		validator.On("ValidateNew", ctx, cluster, rawNewNodePool).Return(nil)

		processor := new(MockNodePoolProcessor)
		processor.On("ProcessNew", ctx, cluster, rawNewNodePool).Return(rawNewNodePool, nil)

		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(
			clusterStore,
			nil,
			clusterGroupManager,
			map[string]Service{
				"eks": distributionService,
			},
			nodePoolStore,
			validator,
			processor,
		)

		err := service.CreateNodePool(ctx, 1, rawNewNodePool)
		require.EqualError(t, err, "create node pool error")

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		rawNewNodePool := NewRawNodePool{
			"name": nodePoolName,
		}

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(false, "", nil)

		distributionService := new(MockService)
		distributionService.On("CreateNodePool", ctx, cluster.ID, rawNewNodePool).Return(nil)

		validator := new(MockNodePoolValidator)
		validator.On("ValidateNew", ctx, cluster, rawNewNodePool).Return(nil)

		processor := new(MockNodePoolProcessor)
		processor.On("ProcessNew", ctx, cluster, rawNewNodePool).Return(rawNewNodePool, nil)

		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(
			clusterStore,
			nil,
			clusterGroupManager,
			map[string]Service{
				"eks": distributionService,
			},
			nodePoolStore,
			validator,
			processor,
		)

		err := service.CreateNodePool(ctx, 1, rawNewNodePool)
		require.NoError(t, err)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})
}

func TestNodePoolService_UpdateNodePool(t *testing.T) {
	t.Run("ClusterNotFound", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)
		{
			err := NotFoundError{ClusterID: 1}
			clusterStore.On("GetCluster", ctx, uint(1)).Return(Cluster{}, err)
		}

		nodePoolStore := new(MockNodePoolStore)
		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		rawNodePoolUpdate := RawNodePoolUpdate{}

		_, err := service.UpdateNodePool(ctx, 1, "pool0", rawNodePoolUpdate)
		require.Error(t, err)

		assert.True(t, errors.Is(err, NotFoundError{ClusterID: 1}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("DistributionNotSupported", func(t *testing.T) {
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
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, map[string]Service{}, nodePoolStore, validator, processor)

		rawNodePoolUpdate := RawNodePoolUpdate{}

		_, err := service.UpdateNodePool(ctx, 1, "pool0", rawNodePoolUpdate)
		require.Error(t, err)

		assert.True(t, errors.As(err, &NotSupportedDistributionError{}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("NodePoolNotFound", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		nodePoolStore := new(MockNodePoolStore)

		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(false, "", nil)

		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		distributions := map[string]Service{
			cluster.Distribution: nil,
		}

		service := NewService(clusterStore, nil, clusterGroupManager, distributions, nodePoolStore, validator, processor)

		rawNodePoolUpdate := RawNodePoolUpdate{}

		_, err := service.UpdateNodePool(ctx, 1, nodePoolName, rawNodePoolUpdate)
		require.Error(t, err)

		assert.True(t, errors.Is(err, NodePoolNotFoundError{ClusterID: 1, NodePool: nodePoolName}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("Failure", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		nodePoolStore := new(MockNodePoolStore)

		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(true, nodePoolName, nil)

		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		distribution := new(MockService)

		rawNodePoolUpdate := RawNodePoolUpdate{}

		distErr := errors.NewPlain("distribution error")

		distribution.On("UpdateNodePool", ctx, cluster.ID, nodePoolName, rawNodePoolUpdate).Return("", distErr)

		distributions := map[string]Service{
			cluster.Distribution: distribution,
		}

		service := NewService(clusterStore, nil, clusterGroupManager, distributions, nodePoolStore, validator, processor)

		_, err := service.UpdateNodePool(ctx, cluster.ID, nodePoolName, rawNodePoolUpdate)
		require.Error(t, err)

		assert.Equal(t, distErr, err)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
		distribution.AssertExpectations(t)
	})

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		nodePoolStore := new(MockNodePoolStore)

		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(true, nodePoolName, nil)

		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		distribution := new(MockService)

		rawNodePoolUpdate := RawNodePoolUpdate{}

		distribution.On("UpdateNodePool", ctx, cluster.ID, nodePoolName, rawNodePoolUpdate).Return("pid", nil)

		distributions := map[string]Service{
			cluster.Distribution: distribution,
		}

		service := NewService(clusterStore, nil, clusterGroupManager, distributions, nodePoolStore, validator, processor)

		_, err := service.UpdateNodePool(ctx, cluster.ID, nodePoolName, rawNodePoolUpdate)
		require.NoError(t, err)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
		distribution.AssertExpectations(t)
	})
}

func TestNodePoolService_DeleteNodePool(t *testing.T) {
	t.Run("ClusterNotFound", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)
		{
			err := NotFoundError{ClusterID: 1}
			clusterStore.On("GetCluster", ctx, uint(1)).Return(Cluster{}, err)
		}

		nodePoolStore := new(MockNodePoolStore)
		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		_, err := service.DeleteNodePool(ctx, 1, "pool0")
		require.Error(t, err)

		assert.True(t, errors.Is(err, NotFoundError{ClusterID: 1}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("DistributionNotSupported", func(t *testing.T) {
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
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		_, err := service.DeleteNodePool(ctx, 1, "pool0")
		require.Error(t, err)

		assert.True(t, errors.As(err, &NotSupportedDistributionError{}))

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("NodePoolExists error", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"
		expectedError := errors.New("test error: NodePoolExists")

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(false, "", expectedError)

		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		deleted, err := service.DeleteNodePool(ctx, 1, nodePoolName)
		require.EqualError(t, err, expectedError.Error())

		assert.False(t, deleted)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("NodePoolDoesNotExist", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(false, "", nil)

		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		deleted, err := service.DeleteNodePool(ctx, 1, nodePoolName)
		require.NoError(t, err)

		assert.True(t, deleted)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("GetDistributionService error", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		nodePoolStore := new(MockNodePoolStore)
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(true, "", nil)

		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(clusterStore, nil, clusterGroupManager, nil, nodePoolStore, validator, processor)

		deleted, err := service.DeleteNodePool(ctx, 1, nodePoolName)
		require.EqualError(t, err, "not supported distribution")

		assert.False(t, deleted)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("NodePoolManager.DeleteNodePool error", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		nodePoolStore := new(MockNodePoolStore)
		storedNodePoolName := "stored-node-pool-name"
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(true, storedNodePoolName, nil)

		distributionService := new(MockService)
		expectedError := errors.New("test error: NodePoolManager.DeleteNodePool")
		distributionService.On("DeleteNodePool", mock.Anything, cluster.ID, storedNodePoolName).Return(false, expectedError)

		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(
			clusterStore,
			nil,
			clusterGroupManager,
			map[string]Service{
				cluster.Distribution: distributionService,
			},
			nodePoolStore,
			validator,
			processor,
		)

		deleted, err := service.DeleteNodePool(ctx, 1, nodePoolName)
		require.EqualError(t, err, expectedError.Error())

		assert.False(t, deleted)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()

		clusterStore := new(MockStore)

		cluster := Cluster{
			ID:            1,
			UID:           "1",
			Name:          "cluster",
			Status:        Running,
			StatusMessage: RunningMessage,
			Cloud:         cloud.Amazon,
			Distribution:  "eks",
		}
		clusterStore.On("GetCluster", ctx, cluster.ID).Return(cluster, nil)

		const nodePoolName = "pool0"

		nodePoolStore := new(MockNodePoolStore)
		storedNodePoolName := "stored-node-pool-name"
		nodePoolStore.On("NodePoolExists", ctx, cluster.ID, nodePoolName).Return(true, storedNodePoolName, nil)

		distributionService := new(MockService)
		distributionService.On("DeleteNodePool", mock.Anything, cluster.ID, storedNodePoolName).Return(true, nil)

		validator := new(MockNodePoolValidator)
		processor := new(MockNodePoolProcessor)
		clusterGroupManager := new(MockClusterGroupManager)

		service := NewService(
			clusterStore,
			nil,
			clusterGroupManager,
			map[string]Service{
				cluster.Distribution: distributionService,
			},
			nodePoolStore,
			validator,
			processor,
		)

		deleted, err := service.DeleteNodePool(ctx, 1, nodePoolName)
		require.NoError(t, err)

		assert.True(t, deleted)

		clusterStore.AssertExpectations(t)
		nodePoolStore.AssertExpectations(t)
		validator.AssertExpectations(t)
		processor.AssertExpectations(t)
	})
}

func TestNodePoolService_ListNodePools(t *testing.T) {
	exampleCluster := Cluster{
		ID:             1,
		UID:            "55956737-a76a-4fb3-a717-f679f1340b41",
		Name:           "cluster-name",
		OrganizationID: 2,
		Status:         "status",
		StatusMessage:  "status message",
		Cloud:          "cloud",
		Distribution:   "eks",
		Location:       "location",
		SecretID:       brn.ResourceName{},
		ConfigSecretID: brn.ResourceName{},
		Tags:           map[string]string{},
	}

	exampleNodePools := RawNodePoolList{
		map[string]interface{}{ // Note: to avoid import cycle of importing eks.NodePool.
			"name": "cluster-node-pool-name-2",
			"labels": map[string]string{
				"label-1": "value-1",
				"label-2": "value-2",
			},
			"size": 4,
			"autoscaling": map[string]interface{}{
				"enabled": true,
				"minSize": 1,
				"maxSize": 2,
			},
			"instanceType": "instance-type",
			"image":        "image",
			"spotPrice":    "5",
		},
		map[string]interface{}{ // Note: to avoid import cycle of importing eks.NodePool.
			"name": "cluster-node-pool-name-3",
			"labels": map[string]string{
				"label-3": "value-3",
			},
			"size": 6,
			"autoscaling": map[string]interface{}{
				"enabled": false,
				"minSize": 0,
				"maxSize": 0,
			},
			"instanceType": "instance-type",
			"image":        "image",
			"spotPrice":    "7",
		},
	}

	type constructionArgumentType struct {
		clusters            Store
		clusterManager      Manager
		clusterGroupManager ClusterGroupManager
		distributions       map[string]Service
		nodePools           NodePoolStore
		nodePoolValidator   NodePoolValidator
		nodePoolProcessor   NodePoolProcessor
	}
	type functionCallArgumentType struct {
		ctx       context.Context
		clusterID uint
	}
	testCases := []struct {
		caseName              string
		constructionArguments constructionArgumentType
		expectedNodePools     RawNodePoolList
		expectedNotNilError   bool
		functionCallArguments functionCallArgumentType
		setupMockFunction     func(constructionArgumentType, functionCallArgumentType)
	}{
		{
			caseName: "StoreGetClusterFailed",
			constructionArguments: constructionArgumentType{
				clusters:            &MockStore{},
				clusterManager:      &MockManager{},
				clusterGroupManager: &MockClusterGroupManager{},
				distributions: map[string]Service{
					"eks": &MockService{},
				},
				nodePools:         &MockNodePoolStore{},
				nodePoolValidator: &MockNodePoolValidator{},
				nodePoolProcessor: &MockNodePoolProcessor{},
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:       context.Background(),
				clusterID: 1,
			},
			setupMockFunction: func(
				constructionArguments constructionArgumentType,
				functionCallArguments functionCallArgumentType,
			) {
				clustersMock := constructionArguments.clusters.(*MockStore)
				clustersMock.On("GetCluster", functionCallArguments.ctx, functionCallArguments.clusterID).Return(Cluster{}, errors.NewPlain("StoreGetClusterFailed"))
			},
		},
		{
			caseName: "ServiceGetDistributionServiceFailed",
			constructionArguments: constructionArgumentType{
				clusters:            &MockStore{},
				clusterManager:      &MockManager{},
				clusterGroupManager: &MockClusterGroupManager{},
				distributions: map[string]Service{
					"eks": &MockService{},
				},
				nodePools:         &MockNodePoolStore{},
				nodePoolValidator: &MockNodePoolValidator{},
				nodePoolProcessor: &MockNodePoolProcessor{},
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:       context.Background(),
				clusterID: 1,
			},
			setupMockFunction: func(
				constructionArguments constructionArgumentType,
				functionCallArguments functionCallArgumentType,
			) {
				unsupportedDistributionCluster := exampleCluster
				unsupportedDistributionCluster.Distribution = "unsupported"

				clustersMock := constructionArguments.clusters.(*MockStore)
				clustersMock.On("GetCluster", functionCallArguments.ctx, functionCallArguments.clusterID).Return(unsupportedDistributionCluster, (error)(nil))
			},
		},
		{
			caseName: "NodePoolServiceListNodePoolsError",
			constructionArguments: constructionArgumentType{
				clusters:            &MockStore{},
				clusterManager:      &MockManager{},
				clusterGroupManager: &MockClusterGroupManager{},
				distributions: map[string]Service{
					"eks": &MockService{},
				},
				nodePools:         &MockNodePoolStore{},
				nodePoolValidator: &MockNodePoolValidator{},
				nodePoolProcessor: &MockNodePoolProcessor{},
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:       context.Background(),
				clusterID: 1,
			},
			setupMockFunction: func(
				constructionArguments constructionArgumentType,
				functionCallArguments functionCallArgumentType,
			) {
				clustersMock := constructionArguments.clusters.(*MockStore)
				clustersMock.On("GetCluster", functionCallArguments.ctx, functionCallArguments.clusterID).Return(exampleCluster, (error)(nil))

				distributionServiceMock := constructionArguments.distributions[exampleCluster.Distribution].(*MockService)
				distributionServiceMock.On(
					"ListNodePools", functionCallArguments.ctx, functionCallArguments.clusterID,
				).Return(nil, errors.New("test error: NodePoolServiceListNodePoolsError"))
			},
		},
		{
			caseName: "NodePoolServiceListNodePoolsSuccess",
			constructionArguments: constructionArgumentType{
				clusters:            &MockStore{},
				clusterManager:      &MockManager{},
				clusterGroupManager: &MockClusterGroupManager{},
				distributions: map[string]Service{
					"eks": &MockService{},
				},
				nodePools:         &MockNodePoolStore{},
				nodePoolValidator: &MockNodePoolValidator{},
				nodePoolProcessor: &MockNodePoolProcessor{},
			},
			expectedNodePools:   exampleNodePools,
			expectedNotNilError: false,
			functionCallArguments: functionCallArgumentType{
				ctx:       context.Background(),
				clusterID: 1,
			},
			setupMockFunction: func(
				constructionArguments constructionArgumentType,
				functionCallArguments functionCallArgumentType,
			) {
				clustersMock := constructionArguments.clusters.(*MockStore)
				clustersMock.On("GetCluster", functionCallArguments.ctx, functionCallArguments.clusterID).Return(exampleCluster, (error)(nil))

				distributionServiceMock := constructionArguments.distributions[exampleCluster.Distribution].(*MockService)
				distributionServiceMock.On("ListNodePools", functionCallArguments.ctx, functionCallArguments.clusterID).Return(exampleNodePools, (error)(nil))
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			testCase.setupMockFunction(testCase.constructionArguments, testCase.functionCallArguments)

			object := service{
				clusters:            testCase.constructionArguments.clusters,
				clusterManager:      testCase.constructionArguments.clusterManager,
				clusterGroupManager: testCase.constructionArguments.clusterGroupManager,
				distributions:       testCase.constructionArguments.distributions,
				nodePools:           testCase.constructionArguments.nodePools,
				nodePoolValidator:   testCase.constructionArguments.nodePoolValidator,
				nodePoolProcessor:   testCase.constructionArguments.nodePoolProcessor,
			}

			actualNodePools, actualError := object.ListNodePools(
				testCase.functionCallArguments.ctx,
				testCase.functionCallArguments.clusterID,
			)

			require.Truef(t, (actualError != nil) == testCase.expectedNotNilError,
				"error value doesn't match the expectation, is expected: %+v, actual error value: %+v", testCase.expectedNotNilError, actualError)
			require.Equal(t, testCase.expectedNodePools, actualNodePools)
		})
	}
}
