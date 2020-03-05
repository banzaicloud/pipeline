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
)

func TestNodePoolValidators_Validate(t *testing.T) {
	ctx := context.Background()
	cluster := Cluster{}
	nodePool := NewRawNodePool{}

	validator1 := new(MockNodePoolValidator)
	validator1.On("ValidateNew", ctx, cluster, nodePool).Return(NewValidationError("invalid node pool", []string{"invalid something"}))

	validator2 := new(MockNodePoolValidator)
	validator2.On("ValidateNew", ctx, cluster, nodePool).Return(errors.New("invalid node pool something"))

	validator3 := new(MockNodePoolValidator)
	validator3.On("ValidateNew", ctx, cluster, nodePool).Return(nil)

	validator := NodePoolValidators{validator1, validator2, validator3}

	err := validator.ValidateNew(ctx, cluster, nodePool)
	require.Error(t, err)

	var verr ValidationError

	assert.True(t, errors.As(err, &verr))
	assert.Equal(
		t,
		[]string{"invalid something", "invalid node pool something"},
		verr.Violations(),
	)

	validator1.AssertExpectations(t)
	validator2.AssertExpectations(t)
	validator3.AssertExpectations(t)
}

func TestNewCommonNodePoolValidator_ValidateNew(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		const labelKey = "key"
		const labelValue = "value"

		labelValidator := new(MockLabelValidator)

		labelValidator.On("ValidateKey", labelKey).Return(nil)
		labelValidator.On("ValidateValue", labelValue).Return(nil)

		nodePool := NewRawNodePool{
			"name": "pool0",
			"labels": map[string]string{
				labelKey: labelValue,
			},
		}

		validator := NewCommonNodePoolValidator(labelValidator)

		err := validator.ValidateNew(context.Background(), Cluster{}, nodePool)
		require.NoError(t, err)

		labelValidator.AssertExpectations(t)
	})

	t.Run("Invalid", func(t *testing.T) {
		const labelKey = "key"
		const labelValue = "value"

		labelValidator := new(MockLabelValidator)

		labelValidator.On("ValidateKey", labelKey).Return(NewValidationError("invalid", []string{"invalid key"}))
		labelValidator.On("ValidateValue", labelValue).Return(NewValidationError("invalid", []string{"invalid value"}))

		nodePool := NewRawNodePool{
			"name": "",
			"labels": map[string]string{
				labelKey: labelValue,
			},
		}

		validator := NewCommonNodePoolValidator(labelValidator)

		err := validator.ValidateNew(context.Background(), Cluster{}, nodePool)
		require.Error(t, err)

		var verr ValidationError

		assert.True(t, errors.As(err, &verr))
		assert.Equal(
			t,
			[]string{"name must be a non-empty string", "invalid key", "invalid value"},
			verr.Violations(),
		)

		labelValidator.AssertExpectations(t)
	})

	t.Run("InvalidSingleLabelError", func(t *testing.T) {
		const labelKey = "key"
		const labelValue = "value"

		labelValidator := new(MockLabelValidator)

		labelValidator.On("ValidateKey", labelKey).Return(errors.New("invalid key"))
		labelValidator.On("ValidateValue", labelValue).Return(errors.New("invalid value"))

		nodePool := NewRawNodePool{
			"name": "",
			"labels": map[string]string{
				labelKey: labelValue,
			},
		}

		validator := NewCommonNodePoolValidator(labelValidator)

		err := validator.ValidateNew(context.Background(), Cluster{}, nodePool)
		require.Error(t, err)

		var verr ValidationError

		assert.True(t, errors.As(err, &verr))
		assert.Equal(
			t,
			[]string{"name must be a non-empty string", "invalid key", "invalid value"},
			verr.Violations(),
		)

		labelValidator.AssertExpectations(t)
	})
}

func TestNewDistributionNodePoolValidator_ValidateNew(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		distValidator := new(MockNodePoolValidator)

		ctx := context.Background()
		cluster := Cluster{
			Distribution: "eks",
		}
		nodePool := NewRawNodePool{}

		distValidator.On("ValidateNew", ctx, cluster, nodePool).Return(nil)

		validator := NewDistributionNodePoolValidator(map[string]NodePoolValidator{
			"eks": distValidator,
		})

		err := validator.ValidateNew(ctx, cluster, nodePool)
		require.NoError(t, err)

		distValidator.AssertExpectations(t)
	})

	t.Run("Invalid", func(t *testing.T) {
		distValidator := new(MockNodePoolValidator)

		ctx := context.Background()
		cluster := Cluster{
			Distribution: "eks",
		}
		nodePool := NewRawNodePool{}

		distErr := errors.New("invalid node pool")

		distValidator.On("ValidateNew", ctx, cluster, nodePool).Return(distErr)

		validator := NewDistributionNodePoolValidator(map[string]NodePoolValidator{
			"eks": distValidator,
		})

		err := validator.ValidateNew(ctx, cluster, nodePool)
		require.Error(t, err)

		assert.Same(t, distErr, err)

		distValidator.AssertExpectations(t)
	})

	t.Run("UnsupportedDistribution", func(t *testing.T) {
		ctx := context.Background()
		cluster := Cluster{
			ID:           1,
			Cloud:        "amazon",
			Distribution: "eks",
		}
		nodePool := NewRawNodePool{}

		validator := NewDistributionNodePoolValidator(map[string]NodePoolValidator{})

		err := validator.ValidateNew(ctx, cluster, nodePool)
		require.Error(t, err)

		expectedErr := NotSupportedDistributionError{
			ID:           cluster.ID,
			Cloud:        cluster.Cloud,
			Distribution: cluster.Distribution,

			Message: "cannot validate unsupported distribution",
		}

		assert.Equal(t, expectedErr, errors.Cause(err))
	})
}
