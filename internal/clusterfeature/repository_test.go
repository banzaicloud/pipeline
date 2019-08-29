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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeFeatureNotFoundError(t *testing.T) {
	assert.True(t, IsFeatureNotFoundError(featureNotFoundError{
		clusterID:   42,
		featureName: "feature",
	}))
}

func TestInmemoryFeatureRepository_GetFeatures(t *testing.T) {
	repository := NewInMemoryFeatureRepository(nil)

	clusterID := uint(1)
	feature := Feature{
		Name: "myFeature",
		Spec: map[string]interface{}{
			"key": "value",
		},
		Status: FeatureStatusActive,
	}

	repository.features[clusterID] = map[string]Feature{
		feature.Name: feature,
	}

	features, err := repository.GetFeatures(context.Background(), clusterID)
	require.NoError(t, err)

	assert.Equal(t, []Feature{feature}, features)
}

func TestInmemoryFeatureRepository_GetFeature(t *testing.T) {
	repository := NewInMemoryFeatureRepository(nil)

	clusterID := uint(1)
	feature := Feature{
		Name: "myFeature",
		Spec: map[string]interface{}{
			"key": "value",
		},
		Status: FeatureStatusActive,
	}

	repository.features[clusterID] = map[string]Feature{
		feature.Name: feature,
	}

	f, err := repository.GetFeature(context.Background(), clusterID, feature.Name)
	require.NoError(t, err)

	assert.Equal(t, feature, f)
}

func TestInmemoryFeatureRepository_SaveFeature(t *testing.T) {
	repository := NewInMemoryFeatureRepository(nil)

	clusterID := uint(1)
	featureName := "myFeature"
	spec := map[string]interface{}{
		"key": "value",
	}

	expectedFeature := Feature{
		Name:   featureName,
		Spec:   spec,
		Status: FeatureStatusPending,
	}

	err := repository.SaveFeature(context.Background(), clusterID, featureName, spec, FeatureStatusPending)
	require.NoError(t, err)

	assert.Equal(t, expectedFeature, repository.features[clusterID][featureName])
}

func TestInmemoryFeatureRepository_DeleteFeature(t *testing.T) {
	repository := NewInMemoryFeatureRepository(nil)

	clusterID := uint(1)
	feature := Feature{
		Name: "myFeature",
		Spec: map[string]interface{}{
			"key": "value",
		},
		Status: FeatureStatusActive,
	}

	repository.features[clusterID] = map[string]Feature{
		feature.Name: feature,
	}

	err := repository.DeleteFeature(context.Background(), clusterID, feature.Name)
	require.NoError(t, err)

	assert.NotContains(t, repository.features[clusterID], feature.Name)
}
