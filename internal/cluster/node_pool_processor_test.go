// Copyright © 2020 Banzai Cloud
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
)

func TestNodePoolProcessors_ProcessNew(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		cluster := Cluster{}
		nodePool := NewRawNodePool{
			"name": "pool0",
		}
		nodePool2 := NewRawNodePool{
			"name": "pool0",
			"key":  "value",
		}

		processor1 := new(MockNodePoolProcessor)
		processor1.On("ProcessNew", ctx, cluster, nodePool).Return(nodePool, nil)

		processor2 := new(MockNodePoolProcessor)
		processor2.On("ProcessNew", ctx, cluster, nodePool).Return(nodePool2, nil)

		processor3 := new(MockNodePoolProcessor)
		processor3.On("ProcessNew", ctx, cluster, nodePool2).Return(nodePool2, nil)

		processor := NodePoolProcessors{processor1, processor2, processor3}

		processedNodePool, err := processor.ProcessNew(ctx, cluster, nodePool)
		require.NoError(t, err)

		assert.Equal(t, nodePool2, processedNodePool)

		processor1.AssertExpectations(t)
		processor2.AssertExpectations(t)
		processor3.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		ctx := context.Background()
		cluster := Cluster{}
		nodePool := NewRawNodePool{
			"name": "pool0",
			"key":  "value",
		}

		processor1 := new(MockNodePoolProcessor)
		perr := errors.New("error")
		processor1.On("ProcessNew", ctx, cluster, nodePool).Return(nodePool, perr)

		processor := NodePoolProcessors{processor1}

		processedNodePool, err := processor.ProcessNew(ctx, cluster, nodePool)
		require.Error(t, err)

		assert.Equal(t, nodePool, processedNodePool)
		assert.Equal(t, perr, err)

		processor1.AssertExpectations(t)
	})
}

func TestCommonNodePoolProcessor_ProcessNew(t *testing.T) {
	ctx := context.Background()
	cluster := Cluster{}
	nodePool := NewRawNodePool{
		"name": "pool0",
		"labels": map[string]string{
			"key":  "value",
			"key2": "value2",
		},
	}

	labelSource := new(MockNodePoolLabelSource)
	labelSource.On("GetLabels", ctx, cluster, nodePool).Return(map[string]string{
		"key2": "value3",
		"key3": "value4",
	}, nil)

	processor := NewCommonNodePoolProcessor(labelSource)

	processedNodePool, err := processor.ProcessNew(ctx, cluster, nodePool)
	require.NoError(t, err)

	assert.Equal(
		t,
		NewRawNodePool{
			"name": "pool0",
			"labels": map[string]string{
				"key":  "value",
				"key2": "value3",
				"key3": "value4",
			},
		},
		processedNodePool,
	)

	labelSource.AssertExpectations(t)
}

func TestNewDistributionNodePoolProcessor_ProcessNew(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		distProcessor := new(MockNodePoolProcessor)

		ctx := context.Background()
		cluster := Cluster{
			Distribution: "eks",
		}
		nodePool := NewRawNodePool{}
		nodePool2 := NewRawNodePool{"key": "value"}

		distProcessor.On("ProcessNew", ctx, cluster, nodePool).Return(nodePool2, nil)

		processor := NewDistributionNodePoolProcessor(map[string]NodePoolProcessor{
			"eks": distProcessor,
		})

		processedNodePool, err := processor.ProcessNew(ctx, cluster, nodePool)
		require.NoError(t, err)

		assert.Equal(t, nodePool2, processedNodePool)

		distProcessor.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		distProcessor := new(MockNodePoolProcessor)

		ctx := context.Background()
		cluster := Cluster{
			Distribution: "eks",
		}
		nodePool := NewRawNodePool{}

		distErr := errors.New("invalid node pool")

		distProcessor.On("ProcessNew", ctx, cluster, nodePool).Return(nodePool, distErr)

		processor := NewDistributionNodePoolProcessor(map[string]NodePoolProcessor{
			"eks": distProcessor,
		})

		_, err := processor.ProcessNew(ctx, cluster, nodePool)
		require.Error(t, err)

		assert.Same(t, distErr, err)

		distProcessor.AssertExpectations(t)
	})

	t.Run("UnsupportedDistribution", func(t *testing.T) {
		ctx := context.Background()
		cluster := Cluster{
			ID:           1,
			Cloud:        "amazon",
			Distribution: "eks",
		}
		nodePool := NewRawNodePool{}

		processor := NewDistributionNodePoolProcessor(map[string]NodePoolProcessor{})

		_, err := processor.ProcessNew(ctx, cluster, nodePool)
		require.Error(t, err)

		expectedErr := NotSupportedDistributionError{
			ID:           cluster.ID,
			Cloud:        cluster.Cloud,
			Distribution: cluster.Distribution,

			Message: "cannot process unsupported distribution",
		}

		assert.Equal(t, expectedErr, errors.Cause(err))
	})
}
