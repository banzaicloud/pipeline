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

package pkeaws

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegionMapImageSelector(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		imageSelector := RegionMapImageSelector{
			"us-east-1": "ami-xxxxxxxxxx",
		}

		image, err := imageSelector.SelectImage(context.Background(), ImageSelectionCriteria{Region: "us-east-1"})
		require.NoError(t, err)

		assert.Equal(t, "ami-xxxxxxxxxx", image)
	})

	t.Run("NoMatches", func(t *testing.T) {
		imageSelector := RegionMapImageSelector{
			"us-east-1": "ami-xxxxxxxxxx",
		}

		image, err := imageSelector.SelectImage(context.Background(), ImageSelectionCriteria{Region: "us-east-2"})

		assert.Equal(t, "", image)
		assert.Equal(t, ImageNotFoundError, errors.Cause(err))
	})
}
