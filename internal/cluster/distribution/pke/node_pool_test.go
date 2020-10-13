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

package pke

import (
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

func TestNodePoolSizeValidation(t *testing.T) {
	base := NewNodePool{
		Name:         "pool",
		InstanceType: "c5.large",
		VolumeSize:   50,
		Image:        "ami",
	}

	t.Run("PoolZeroSize", func(t *testing.T) {
		pool := base
		pool.Size = 0
		pool.Autoscaling.Enabled = false

		err := pool.Validate()
		assert.IsType(t, cluster.ValidationError{}, errors.Cause(err))
	})

	t.Run("PoolMinSizeNegative", func(t *testing.T) {
		pool := base
		pool.Size = 1
		pool.Autoscaling.Enabled = true
		pool.Autoscaling.MinSize = -1
		pool.Autoscaling.MaxSize = 2

		err := pool.Validate()
		assert.IsType(t, cluster.ValidationError{}, errors.Cause(err))
	})

	t.Run("PoolSizeUnderMin", func(t *testing.T) {
		pool := base
		pool.Size = 0
		pool.Autoscaling.Enabled = true
		pool.Autoscaling.MinSize = 1
		pool.Autoscaling.MaxSize = 2

		err := pool.Validate()
		assert.IsType(t, cluster.ValidationError{}, errors.Cause(err))
	})

	t.Run("PoolSizeOverMax", func(t *testing.T) {
		pool := base
		pool.Size = 3
		pool.Autoscaling.Enabled = true
		pool.Autoscaling.MinSize = 1
		pool.Autoscaling.MaxSize = 2

		err := pool.Validate()
		assert.IsType(t, cluster.ValidationError{}, errors.Cause(err))
	})

	t.Run("PoolReverseRange", func(t *testing.T) {
		pool := base
		pool.Size = 1
		pool.Autoscaling.Enabled = true
		pool.Autoscaling.MinSize = 2
		pool.Autoscaling.MaxSize = 1

		err := pool.Validate()
		assert.IsType(t, cluster.ValidationError{}, errors.Cause(err))
	})

	t.Run("PoolValidRange", func(t *testing.T) {
		pool := base
		pool.Size = 1
		pool.Autoscaling.Enabled = true
		pool.Autoscaling.MinSize = 1
		pool.Autoscaling.MaxSize = 2

		err := pool.Validate()
		assert.NoError(t, err)
	})

	t.Run("PoolZeroAutoscaling", func(t *testing.T) {
		pool := base
		pool.Size = 0
		pool.Autoscaling.Enabled = true
		pool.Autoscaling.MinSize = 0
		pool.Autoscaling.MaxSize = 2

		err := pool.Validate()
		assert.NoError(t, err)
	})

	t.Run("PoolValidNoAutoscaling", func(t *testing.T) {
		pool := base
		pool.Size = 1
		pool.Autoscaling.Enabled = false

		err := pool.Validate()
		assert.NoError(t, err)
	})
}
