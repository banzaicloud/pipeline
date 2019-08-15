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

	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
)

func TestFeatureService_List(t *testing.T) {
	repository := NewInMemoryFeatureRepository()
	featureManager := NewSyncFeatureManager(&dummyFeatureManager{}, repository, commonadapter.NewNoopLogger())
	registry := NewFeatureRegistry(map[string]FeatureManager{
		featureManager.Name(): featureManager,
	})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())

	clusterID := uint(1)
	featureName := featureManager.Name()
	spec := FeatureSpec{"key": "value"}

	feature := Feature{
		Name:   featureName,
		Spec:   spec,
		Status: FeatureStatusActive,
		Output: map[string]interface{}{},
	}

	repository.features[clusterID] = map[string]Feature{
		featureName: feature,
	}

	expectedFeatures := []Feature{feature}

	features, err := service.List(context.Background(), clusterID)
	require.NoError(t, err)

	assert.Equal(t, expectedFeatures, features)
}

func TestFeatureService_Details(t *testing.T) {
	repository := NewInMemoryFeatureRepository()
	featureManager := NewSyncFeatureManager(&dummyFeatureManager{}, repository, commonadapter.NewNoopLogger())
	registry := NewFeatureRegistry(map[string]FeatureManager{
		featureManager.Name(): featureManager,
	})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())

	clusterID := uint(1)
	featureName := featureManager.Name()
	spec := FeatureSpec{"key": "value"}

	feature := Feature{
		Name:   featureName,
		Spec:   spec,
		Status: FeatureStatusActive,
	}

	repository.features[clusterID] = map[string]Feature{
		featureName: feature,
	}

	expectedFeature := &Feature{
		Name: "myFeature",
		Spec: FeatureSpec{
			"key": "value",
		},
		Output: map[string]interface{}{},
		Status: FeatureStatusActive,
	}

	f, err := service.Details(context.Background(), clusterID, featureName)
	require.NoError(t, err)

	assert.Equal(t, expectedFeature, f)
}

func TestFeatureService_Activate(t *testing.T) {
	repository := NewInMemoryFeatureRepository()
	featureManager := NewSyncFeatureManager(&dummyFeatureManager{}, repository, commonadapter.NewNoopLogger())
	registry := NewFeatureRegistry(map[string]FeatureManager{
		featureManager.Name(): featureManager,
	})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())

	clusterID := uint(1)
	featureName := featureManager.Name()
	spec := FeatureSpec{"key": "value"}

	err := service.Activate(context.Background(), clusterID, featureName, spec)
	require.NoError(t, err)

	feature, err := repository.GetFeature(context.Background(), clusterID, featureName)
	require.NoError(t, err)

	assert.Equal(t, featureName, feature.Name)
	assert.Equal(t, spec["key"], feature.Spec["key"])
	assert.Equal(t, FeatureStatusActive, feature.Status)
}

func TestFeatureService_Activate_UnknownFeature(t *testing.T) {
	repository := NewInMemoryFeatureRepository()
	registry := NewFeatureRegistry(map[string]FeatureManager{})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())

	clusterID := uint(1)
	featureName := "notMyFeature"
	spec := FeatureSpec{"key": "value"}

	err := service.Activate(context.Background(), clusterID, featureName, spec)
	require.Error(t, err)

	assert.True(t, errors.As(err, &UnknownFeatureError{}))
	assert.True(t, errors.Is(err, UnknownFeatureError{FeatureName: featureName}))

	feature, err := repository.GetFeature(context.Background(), clusterID, featureName)
	require.NoError(t, err)

	assert.Nil(t, feature)
}

func TestFeatureService_Activate_FeatureAlreadyActivated(t *testing.T) {
	t.Skip("implement async feature manager")
	repository := NewInMemoryFeatureRepository()
	featureManager := NewSyncFeatureManager(&dummyFeatureManager{}, repository, commonadapter.NewNoopLogger())
	registry := NewFeatureRegistry(map[string]FeatureManager{
		featureManager.Name(): featureManager,
	})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())

	clusterID := uint(1)
	featureName := featureManager.Name()
	spec := FeatureSpec{"key": "value"}

	repository.features[clusterID] = map[string]Feature{
		featureName: {
			Name:   featureName,
			Spec:   spec,
			Status: FeatureStatusActive,
		},
	}

	err := service.Activate(context.Background(), clusterID, featureName, spec)
	require.Error(t, err)

	assert.True(t, errors.As(err, &FeatureAlreadyActivatedError{}))
	assert.True(t, errors.Is(err, FeatureAlreadyActivatedError{FeatureName: featureName}))
}

func TestFeatureService_Activate_InvalidSpec(t *testing.T) {
	repository := NewInMemoryFeatureRepository()
	featureManager := NewSyncFeatureManager(&dummyFeatureManager{}, repository, commonadapter.NewNoopLogger())
	registry := NewFeatureRegistry(map[string]FeatureManager{
		featureManager.Name(): featureManager,
	})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())

	clusterID := uint(1)
	featureName := featureManager.Name()
	spec := FeatureSpec{}

	err := service.Activate(context.Background(), clusterID, featureName, spec)
	require.Error(t, err)

	assert.True(t, errors.As(err, &InvalidFeatureSpecError{}))
	assert.True(t, errors.Is(err, InvalidFeatureSpecError{FeatureName: featureName, Problem: "invalid feature spec: key should have value"}))

	feature, err := repository.GetFeature(context.Background(), clusterID, featureName)
	require.NoError(t, err)

	assert.Nil(t, feature)
}

func TestFeatureService_Activate_ActivationFails(t *testing.T) {
	repository := NewInMemoryFeatureRepository()
	featureManager := NewSyncFeatureManager(&dummyFeatureManager{}, repository, commonadapter.NewNoopLogger())
	registry := NewFeatureRegistry(map[string]FeatureManager{
		featureManager.Name(): featureManager,
	})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())

	clusterID := uint(1)
	featureName := featureManager.Name()
	spec := FeatureSpec{"key": "value", "fail": true}

	err := service.Activate(context.Background(), clusterID, featureName, spec)
	require.Error(t, err)

	feature, err := repository.GetFeature(context.Background(), clusterID, featureName)
	require.NoError(t, err)

	assert.Nil(t, feature)
}

func TestFeatureService_Deactivate(t *testing.T) {
	repository := NewInMemoryFeatureRepository()
	featureManager := NewSyncFeatureManager(&dummyFeatureManager{}, repository, commonadapter.NewNoopLogger())
	registry := NewFeatureRegistry(map[string]FeatureManager{
		featureManager.Name(): featureManager,
	})

	clusterID := uint(2)
	featureName := featureManager.Name()
	spec := FeatureSpec{"key": "value", "fail": true}

	// a persisted, active feature
	repository.CreateFeature(context.Background(), clusterID, featureName, spec, FeatureStatusActive) // nolint: errcheck

	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())

	err := service.Deactivate(context.Background(), clusterID, featureName)
	require.NoError(t, err)
}

// TestFeatureService_Deactivate_NotActive (not found in the persistent store)
func TestFeatureService_Deactivate_NotActive(t *testing.T) {
	t.Skip("implement async feature manager")
	repository := NewInMemoryFeatureRepository()
	featureManager := NewSyncFeatureManager(&dummyFeatureManager{}, repository, commonadapter.NewNoopLogger())
	registry := NewFeatureRegistry(map[string]FeatureManager{
		featureManager.Name(): featureManager,
	})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())
	featureName := featureManager.Name()
	err := service.Deactivate(context.Background(), 1, featureName)
	require.Error(t, err)

	assert.True(t, errors.As(err, &FeatureNotActiveError{}))
	assert.True(t, errors.Is(err, FeatureNotActiveError{FeatureName: featureName}))
}

// TestFeatureService_Deactivate_UnknownFeature no manager registered for this feature name
func TestFeatureService_Deactivate_UnknownFeature(t *testing.T) {
	repository := NewInMemoryFeatureRepository()
	registry := NewFeatureRegistry(map[string]FeatureManager{})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())
	clusterID := uint(1)
	featureName := "notMyFeature"
	repository.CreateFeature(context.Background(), clusterID, featureName, FeatureSpec{"persisted": "feature"}, FeatureStatusActive) // nolint: errcheck

	err := service.Deactivate(context.Background(), clusterID, featureName)
	require.Error(t, err)

	assert.True(t, errors.As(err, &UnknownFeatureError{}))
	assert.True(t, errors.Is(err, UnknownFeatureError{FeatureName: featureName}))
}

func TestFeatureService_Deactivate_DeactivationFails(t *testing.T) {
	repository := NewInMemoryFeatureRepository()
	featureManager := NewSyncFeatureManager(&dummyFeatureManager{}, repository, commonadapter.NewNoopLogger())
	registry := NewFeatureRegistry(map[string]FeatureManager{
		featureManager.Name(): featureManager,
	})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())
	clusterID := uint(1)
	featureName := featureManager.Name()
	repository.CreateFeature(context.Background(), clusterID, featureName, FeatureSpec{"key": "value"}, FeatureStatusActive) // nolint: errcheck
	err := service.Deactivate(context.Background(), clusterID, featureName)

	// do we need a specific error for this?
	require.Error(t, err)
}

func TestFeatureService_Update(t *testing.T) {
	repository := NewInMemoryFeatureRepository()
	featureManager := NewSyncFeatureManager(&dummyFeatureManager{}, repository, commonadapter.NewNoopLogger())
	registry := NewFeatureRegistry(map[string]FeatureManager{
		featureManager.Name(): featureManager,
	})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())

	clusterID := uint(1)
	featureName := featureManager.Name()
	spec := FeatureSpec{"key": "value"}
	repository.CreateFeature(context.Background(), clusterID, featureName, spec, FeatureStatusActive) // nolint: errcheck

	err := service.Update(context.Background(), clusterID, featureName, spec)
	require.NoError(t, err)
}

func TestFeatureService_Update_NotActive(t *testing.T) {
	t.Skip("implement async feature manager")
	repository := NewInMemoryFeatureRepository()
	featureManager := NewSyncFeatureManager(&dummyFeatureManager{}, repository, commonadapter.NewNoopLogger())
	registry := NewFeatureRegistry(map[string]FeatureManager{
		featureManager.Name(): featureManager,
	})

	spec := FeatureSpec{"key": "value", "fail": true}

	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())
	featureName := featureManager.Name()
	err := service.Update(context.Background(), 1, featureName, spec)
	require.Error(t, err)

	assert.True(t, errors.As(err, &FeatureNotActiveError{}))
	assert.True(t, errors.Is(err, FeatureNotActiveError{FeatureName: featureName}))

}

func TestFeatureService_Update_UnknownFeature(t *testing.T) {
	repository := NewInMemoryFeatureRepository()

	registry := NewFeatureRegistry(map[string]FeatureManager{})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())

	clusterID := uint(1)
	featureName := "notMyFeature"
	spec := FeatureSpec{"key": "value"}

	repository.CreateFeature(context.Background(), clusterID, featureName, FeatureSpec{"persisted": "feature"}, FeatureStatusActive) // nolint: errcheck

	err := service.Update(context.Background(), clusterID, featureName, spec)
	require.Error(t, err)

	assert.True(t, errors.As(err, &UnknownFeatureError{}))
	assert.True(t, errors.Is(err, UnknownFeatureError{FeatureName: featureName}))

}

func TestFeatureService_Update_InvalidSpec(t *testing.T) {
	repository := NewInMemoryFeatureRepository()
	featureManager := NewSyncFeatureManager(&dummyFeatureManager{}, repository, commonadapter.NewNoopLogger())
	registry := NewFeatureRegistry(map[string]FeatureManager{
		featureManager.Name(): featureManager,
	})
	service := NewFeatureService(registry, repository, commonadapter.NewNoopLogger())

	clusterID := uint(1)
	featureName := featureManager.Name()
	spec := FeatureSpec{}

	repository.CreateFeature(context.Background(), clusterID, featureName, FeatureSpec{"persisted": "feature"}, FeatureStatusActive) // nolint: errcheck

	err := service.Update(context.Background(), clusterID, featureName, spec)
	require.Error(t, err)

	assert.True(t, errors.As(err, &InvalidFeatureSpecError{}))
	assert.True(t, errors.Is(err, InvalidFeatureSpecError{FeatureName: featureName, Problem: "invalid feature spec: key should have value"}))

}

func TestFeatureService_Update_UpdateFails(t *testing.T) {
	// TODO(laszlop): write tests for this scenario
	// TODO(laszlop): specify the expected behavior
}
