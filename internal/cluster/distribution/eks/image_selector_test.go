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

package eks

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestKubernetesVersionImageSelector(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		constraint, err := semver.NewConstraint("~1.17")
		require.NoError(t, err)

		criteria := ImageSelectionCriteria{
			Region:            "us-east-1",
			KubernetesVersion: "1.17.7",
		}

		delegatedImageSelector := new(MockImageSelector)
		delegatedImageSelector.On("SelectImage", mock.Anything, criteria).Return("ami-xxxxxxxxxx", nil)

		imageSelector := KubernetesVersionImageSelector{
			Constraint:    constraint,
			ImageSelector: delegatedImageSelector,
		}

		image, err := imageSelector.SelectImage(context.Background(), criteria)
		require.NoError(t, err)

		assert.Equal(t, "ami-xxxxxxxxxx", image)
		delegatedImageSelector.AssertExpectations(t)
	})

	t.Run("ConstraintDoesNotMatch", func(t *testing.T) {
		constraint, err := semver.NewConstraint("~1.17")
		require.NoError(t, err)

		criteria := ImageSelectionCriteria{
			Region:            "us-east-1",
			KubernetesVersion: "1.16.6",
		}

		imageSelector := KubernetesVersionImageSelector{
			Constraint: constraint,
		}

		image, err := imageSelector.SelectImage(context.Background(), criteria)
		require.Error(t, err)

		assert.Equal(t, "", image)
		assert.Equal(t, ImageNotFoundError, errors.Cause(err))
	})
}

func TestImageSelectors(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		criteria := ImageSelectionCriteria{
			Region: "us-east-1",
		}

		imageSelector1 := new(MockImageSelector)
		imageSelector1.On("SelectImage", mock.Anything, criteria).Return("ami-xxxxxxxxxx", nil)

		imageSelector := ImageSelectors{imageSelector1}

		image, err := imageSelector.SelectImage(context.Background(), criteria)
		require.NoError(t, err)

		assert.Equal(t, "ami-xxxxxxxxxx", image)
		imageSelector1.AssertExpectations(t)
	})

	t.Run("Empty", func(t *testing.T) {
		criteria := ImageSelectionCriteria{
			Region: "us-east-1",
		}

		imageSelector := ImageSelectors{}

		image, err := imageSelector.SelectImage(context.Background(), criteria)
		require.Error(t, err)

		assert.Equal(t, "", image)
		assert.Equal(t, ImageNotFoundError, errors.Cause(err))
	})

	t.Run("FallbackIfNotFound", func(t *testing.T) {
		criteria := ImageSelectionCriteria{
			Region: "us-east-1",
		}

		imageSelector1 := new(MockImageSelector)
		imageSelector1.On("SelectImage", mock.Anything, criteria).Return("", errors.Wrap(ImageNotFoundError, "selector1"))

		imageSelector2 := new(MockImageSelector)
		imageSelector2.On("SelectImage", mock.Anything, criteria).Return("ami-xxxxxxxxxx", nil)

		imageSelector := ImageSelectors{imageSelector1, imageSelector2}

		image, err := imageSelector.SelectImage(context.Background(), criteria)
		require.NoError(t, err)

		assert.Equal(t, "ami-xxxxxxxxxx", image)
		imageSelector1.AssertExpectations(t)
		imageSelector2.AssertExpectations(t)
	})

	t.Run("FallbackIfError", func(t *testing.T) {
		criteria := ImageSelectionCriteria{
			Region: "us-east-1",
		}

		imageSelector1 := new(MockImageSelector)
		imageSelector1.On("SelectImage", mock.Anything, criteria).Return("", errors.New("fatal error"))

		imageSelector2 := new(MockImageSelector)
		imageSelector2.On("SelectImage", mock.Anything, criteria).Return("ami-xxxxxxxxxx", nil)

		imageSelector := ImageSelectors{imageSelector1, imageSelector2}

		image, err := imageSelector.SelectImage(context.Background(), criteria)
		require.NoError(t, err)

		assert.Equal(t, "ami-xxxxxxxxxx", image)
		imageSelector1.AssertExpectations(t)
		imageSelector2.AssertExpectations(t)
	})
}

func TestDefaultImageSelector(t *testing.T) {
	t.Run("GPU", func(t *testing.T) {
		criteria := ImageSelectionCriteria{
			Region:       "us-east-1",
			InstanceType: "p2.xlarge",
		}

		defaultAcceleratedImageSelector := new(MockImageSelector)
		defaultAcceleratedImageSelector.On("SelectImage", mock.Anything, criteria).Return("ami-xxxxxxxxxx", nil)

		imageSelector := DefaultImageSelector{
			DefaultImages:            nil,
			DefaultAcceleratedImages: defaultAcceleratedImageSelector,
		}

		image, err := imageSelector.SelectImage(context.Background(), criteria)
		require.NoError(t, err)

		assert.Equal(t, "ami-xxxxxxxxxx", image)
		defaultAcceleratedImageSelector.AssertExpectations(t)
	})

	t.Run("GPUNotFound", func(t *testing.T) {
		criteria := ImageSelectionCriteria{
			Region:       "us-east-1",
			InstanceType: "p2.xlarge",
		}

		defaultAcceleratedImageSelector := new(MockImageSelector)
		defaultAcceleratedImageSelector.On("SelectImage", mock.Anything, criteria).Return("", ImageNotFoundError)

		defaultImageSelector := new(MockImageSelector)
		defaultImageSelector.On("SelectImage", mock.Anything, criteria).Return("ami-xxxxxxxxxx", nil)

		imageSelector := DefaultImageSelector{
			DefaultImages:            defaultImageSelector,
			DefaultAcceleratedImages: defaultAcceleratedImageSelector,
		}

		image, err := imageSelector.SelectImage(context.Background(), criteria)
		require.NoError(t, err)

		assert.Equal(t, "ami-xxxxxxxxxx", image)
		defaultAcceleratedImageSelector.AssertExpectations(t)
		defaultImageSelector.AssertExpectations(t)
	})

	t.Run("NonGPU", func(t *testing.T) {
		criteria := ImageSelectionCriteria{
			Region:       "us-east-1",
			InstanceType: "t2.medium",
		}

		defaultImageSelector := new(MockImageSelector)
		defaultImageSelector.On("SelectImage", mock.Anything, criteria).Return("ami-xxxxxxxxxx", nil)

		imageSelector := DefaultImageSelector{
			DefaultImages:            defaultImageSelector,
			DefaultAcceleratedImages: nil,
		}

		image, err := imageSelector.SelectImage(context.Background(), criteria)
		require.NoError(t, err)

		assert.Equal(t, "ami-xxxxxxxxxx", image)
		defaultImageSelector.AssertExpectations(t)
	})

	t.Run("ARM", func(t *testing.T) {
		criteria := ImageSelectionCriteria{
			Region:       "us-east-1",
			InstanceType: "a1.xlarge",
		}

		defaultARMImageSelector := new(MockImageSelector)
		defaultARMImageSelector.On("SelectImage", mock.Anything, criteria).Return("ami-xxxxxxxxxx", nil)

		imageSelector := DefaultImageSelector{
			DefaultImages:    nil,
			DefaultARMImages: defaultARMImageSelector,
		}

		image, err := imageSelector.SelectImage(context.Background(), criteria)
		require.NoError(t, err)

		assert.Equal(t, "ami-xxxxxxxxxx", image)
		defaultARMImageSelector.AssertExpectations(t)
	})
}
