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

package clusterfeature

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureManagerRegistry_GetFeatureManager(t *testing.T) {
	expectedFeatureManager := dummyFeatureManager{
		TheName: "myFeature",
	}

	registry := MakeFeatureManagerRegistry([]FeatureManager{
		expectedFeatureManager,
	})

	featureManager, err := registry.GetFeatureManager("myFeature")
	require.NoError(t, err)

	assert.Equal(t, expectedFeatureManager, featureManager)
}

func TestFeatureManagerRegistry_GetFeatureManager_UnknownFeature(t *testing.T) {
	registry := MakeFeatureManagerRegistry([]FeatureManager{})

	featureManager, err := registry.GetFeatureManager("myFeature")
	require.Error(t, err)

	assert.True(t, errors.As(err, &UnknownFeatureError{}))
	assert.True(t, errors.Is(err, UnknownFeatureError{FeatureName: "myFeature"}))

	assert.Nil(t, featureManager)
}

func TestFeatureOperatorRegistry_GetFeatureOperator(t *testing.T) {
	expectedFeatureOperator := dummyFeatureOperator{
		TheName: "myFeature",
	}

	registry := MakeFeatureOperatorRegistry([]FeatureOperator{
		expectedFeatureOperator,
	})

	featureOperator, err := registry.GetFeatureOperator("myFeature")
	require.NoError(t, err)

	assert.Equal(t, expectedFeatureOperator, featureOperator)
}

func TestFeatureOperatorRegistry_GetFeatureOperator_UnknownFeature(t *testing.T) {
	registry := MakeFeatureOperatorRegistry([]FeatureOperator{})

	featureOperator, err := registry.GetFeatureOperator("myFeature")
	require.Error(t, err)

	assert.True(t, errors.As(err, &UnknownFeatureError{}))
	assert.True(t, errors.Is(err, UnknownFeatureError{FeatureName: "myFeature"}))

	assert.Nil(t, featureOperator)
}

type dummyFeatureManager struct {
	TheName         string
	Output          FeatureOutput
	ValidationError error
}

func (d dummyFeatureManager) Name() string {
	return d.TheName
}

func (d dummyFeatureManager) GetOutput(ctx context.Context, clusterID uint) (FeatureOutput, error) {
	return d.Output, nil
}

func (d dummyFeatureManager) ValidateSpec(ctx context.Context, spec FeatureSpec) error {
	return d.ValidationError
}

func (d dummyFeatureManager) PrepareSpec(ctx context.Context, spec FeatureSpec) (FeatureSpec, error) {
	return spec, nil
}

type dummyFeatureOperator struct {
	TheName string
}

func (d dummyFeatureOperator) Name() string {
	return d.TheName
}

func (d dummyFeatureOperator) Apply(ctx context.Context, clusterID uint, spec FeatureSpec) error {
	return nil
}

func (d dummyFeatureOperator) Deactivate(ctx context.Context, clusterID uint) error {
	return nil
}
