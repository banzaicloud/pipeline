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

func TestNodePoolLabelSources_GetLabels(t *testing.T) {
	ctx := context.Background()
	cluster := Cluster{}
	nodePool := NodePool(nil)

	source1 := new(MockNodePoolLabelSource)
	source1.On("GetLabels", ctx, cluster, nodePool).Return(map[string]string{"key": "value"}, nil)

	source2 := new(MockNodePoolLabelSource)
	source2.On("GetLabels", ctx, cluster, nodePool).Return(map[string]string{"key": "value2"}, nil)

	source3 := new(MockNodePoolLabelSource)
	source3.On("GetLabels", ctx, cluster, nodePool).Return(map[string]string{"key2": "value3"}, nil)

	source4 := new(MockNodePoolLabelSource)
	err4 := errors.New("invalid node pool something")
	source4.On("GetLabels", ctx, cluster, nodePool).Return(nil, err4)

	source := NodePoolLabelSources{source1, source2, source3, source4}

	labels, err := source.GetLabels(ctx, cluster, nodePool)

	assert.Equal(t, map[string]string{"key": "value2", "key2": "value3"}, labels)
	assert.Equal(t, []error{err4}, errors.GetErrors(err))

	source1.AssertExpectations(t)
	source2.AssertExpectations(t)
	source3.AssertExpectations(t)
	source4.AssertExpectations(t)
}

func TestCommonNodePoolLabelSource_GetLabels(t *testing.T) {
	nodePool := nodePoolStub{
		name:     "pool0",
		onDemand: true,
	}

	source := NewCommonNodePoolLabelSource()

	labels, err := source.GetLabels(context.Background(), Cluster{}, nodePool)
	require.NoError(t, err)

	assert.Equal(
		t,
		map[string]string{
			NodePoolNameLabelKey: "pool0",
			nodeOnDemandLabelKey: "true",
		},
		labels,
	)
}

func TestFilterValidNodePoolLabelSource_GetLabels(t *testing.T) {
	nodePool := nodePoolStub{
		labels: map[string]string{
			"key":  "value",
			"key2": "value2",
		},
	}

	labelValidator := new(MockLabelValidator)
	labelValidator.On("ValidateKey", "key").Return(nil)
	labelValidator.On("ValidateValue", "value").Return(nil)
	labelValidator.On("ValidateKey", "key2").Return(errors.New("invalid key"))

	source := NewFilterValidNodePoolLabelSource(labelValidator)

	labels, err := source.GetLabels(context.Background(), Cluster{}, nodePool)
	require.NoError(t, err)

	assert.Equal(
		t,
		map[string]string{
			"key": "value",
		},
		labels,
	)

	labelValidator.AssertExpectations(t)
}
