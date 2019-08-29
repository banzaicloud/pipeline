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

	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
)

func TestFeatureService_List(t *testing.T) {
	clusterID := uint(1)
	expectedFeatures := []Feature{
		{
			Name: "myActiveFeature",
			Spec: FeatureSpec{
				"someSpecKey": "someSpecValue",
			},
			Output: FeatureOutput{
				"someOutputKey": "someOutputValue",
			},
			Status: FeatureStatusActive,
		},
		{
			Name: "myPendingFeature",
			Spec: FeatureSpec{
				"mySpecKey": "mySpecValue",
			},
			Output: FeatureOutput{
				"myOutputKey": "myOutputValue",
			},
			Status: FeatureStatusPending,
		},
	}
	featureManagers := make([]FeatureManager, len(expectedFeatures))
	storedFeatures := make([]Feature, len(expectedFeatures))
	for i, f := range expectedFeatures {
		featureManagers[i] = &dummyFeatureManager{
			TheName: f.Name,
			Output:  f.Output,
		}

		storedFeatures[i] = Feature{
			Name:   f.Name,
			Spec:   f.Spec,
			Status: f.Status,
		}
	}
	registry := MakeFeatureManagerRegistry(featureManagers)
	repository := NewInMemoryFeatureRepository(map[uint][]Feature{
		clusterID: storedFeatures,
	})
	logger := commonadapter.NewNoopLogger()
	service := MakeFeatureService(nil, registry, repository, logger)

	features, err := service.List(context.Background(), clusterID)
	assert.NoError(t, err)
	assert.ElementsMatch(t, expectedFeatures, features)
}

func TestFeatureService_Details(t *testing.T) {
	clusterID := uint(1)
	featureName := "myFeature"
	expectedFeature := Feature{
		Name: featureName,
		Spec: FeatureSpec{
			"mySpecKey": "mySpecValue",
		},
		Output: FeatureOutput{
			"myOutputKey": "myOutputValue",
		},
		Status: FeatureStatusActive,
	}
	registry := MakeFeatureManagerRegistry([]FeatureManager{
		&dummyFeatureManager{
			TheName: expectedFeature.Name,
			Output:  expectedFeature.Output,
		},
	})
	repository := NewInMemoryFeatureRepository(map[uint][]Feature{
		clusterID: {
			{
				Name:   expectedFeature.Name,
				Spec:   expectedFeature.Spec,
				Status: expectedFeature.Status,
			},
		},
	})
	logger := commonadapter.NewNoopLogger()
	service := MakeFeatureService(nil, registry, repository, logger)

	feature, err := service.Details(context.Background(), clusterID, featureName)
	assert.NoError(t, err)
	assert.Equal(t, expectedFeature, feature)
}

func TestFeatureService_Activate(t *testing.T) {
	clusterID := uint(1)
	featureName := "myFeature"
	dispatcher := &dummyFeatureOperationDispatcher{}
	featureManager := &dummyFeatureManager{
		TheName: featureName,
		Output: FeatureOutput{
			"someKey": "someValue",
		},
	}
	registry := MakeFeatureManagerRegistry([]FeatureManager{featureManager})
	repository := NewInMemoryFeatureRepository(nil)
	logger := commonadapter.NewNoopLogger()
	service := MakeFeatureService(dispatcher, registry, repository, logger)

	cases := map[string]struct {
		FeatureName     string
		ValidationError error
		ApplyError      error
		Error           interface{}
		FeatureSaved    bool
	}{
		"success": {
			FeatureName:  featureName,
			FeatureSaved: true,
		},
		"unknown feature": {
			FeatureName: "notMyFeature",
			Error: UnknownFeatureError{
				FeatureName: "notMyFeature",
			},
		},
		"invalid spec": {
			FeatureName:     featureName,
			ValidationError: errors.New("validation error"),
			Error:           true,
		},
		"begin apply fails": {
			FeatureName: featureName,
			ApplyError:  errors.New("failed to begin apply"),
			Error:       true,
		},
	}
	spec := FeatureSpec{
		"mySpecKey": "mySpecValue",
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repository.Clear()
			dispatcher.ApplyError = tc.ApplyError
			featureManager.ValidationError = tc.ValidationError

			err := service.Activate(context.Background(), clusterID, tc.FeatureName, spec)
			switch tc.Error {
			case true:
				assert.Error(t, err)
			case nil, false:
				assert.NoError(t, err)
			default:
				assert.Equal(t, tc.Error, errors.Cause(err))
			}

			if tc.FeatureSaved {
				assert.NotEmpty(t, repository.features[clusterID])
			} else {
				assert.Empty(t, repository.features[clusterID])
			}
		})
	}
}

func TestFeatureService_Deactivate(t *testing.T) {
	clusterID := uint(1)
	featureName := "myFeature"
	dispatcher := &dummyFeatureOperationDispatcher{}
	registry := MakeFeatureManagerRegistry([]FeatureManager{
		dummyFeatureManager{
			TheName: featureName,
			Output: FeatureOutput{
				"someKey": "someValue",
			},
		},
	})
	repository := NewInMemoryFeatureRepository(map[uint][]Feature{
		clusterID: {
			{
				Name: featureName,
				Spec: FeatureSpec{
					"mySpecKey": "mySpecValue",
				},
				Status: FeatureStatusActive,
			},
		},
	})
	snapshot := repository.Snapshot()
	logger := commonadapter.NewNoopLogger()
	service := MakeFeatureService(dispatcher, registry, repository, logger)

	cases := map[string]struct {
		FeatureName     string
		DeactivateError error
		Error           interface{}
		StatusAfter     FeatureStatus
	}{
		"success": {
			FeatureName: featureName,
			StatusAfter: FeatureStatusPending,
		},
		"unknown feature": {
			FeatureName: "notMyFeature",
			Error: UnknownFeatureError{
				FeatureName: "notMyFeature",
			},
			StatusAfter: FeatureStatusActive,
		},
		"begin deactivate fails": {
			FeatureName:     featureName,
			DeactivateError: errors.New("failed to begin deactivate"),
			Error:           true,
			StatusAfter:     FeatureStatusActive,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repository.Restore(snapshot)
			dispatcher.DeactivateError = tc.DeactivateError

			err := service.Deactivate(context.Background(), clusterID, tc.FeatureName)
			switch tc.Error {
			case true:
				assert.Error(t, err)
			case nil, false:
				assert.NoError(t, err)
			default:
				assert.Equal(t, tc.Error, errors.Cause(err))
			}

			assert.Equal(t, tc.StatusAfter, repository.features[clusterID][featureName].Status)
		})
	}
}

func TestFeatureService_Update(t *testing.T) {
	clusterID := uint(1)
	featureName := "myFeature"
	dispatcher := &dummyFeatureOperationDispatcher{}
	featureManager := &dummyFeatureManager{
		TheName: featureName,
		Output: FeatureOutput{
			"someKey": "someValue",
		},
	}
	registry := MakeFeatureManagerRegistry([]FeatureManager{featureManager})
	repository := NewInMemoryFeatureRepository(map[uint][]Feature{
		clusterID: {
			{
				Name: featureName,
				Spec: FeatureSpec{
					"mySpecKey": "mySpecValue",
				},
				Status: FeatureStatusActive,
			},
		},
	})
	snapshot := repository.Snapshot()
	logger := commonadapter.NewNoopLogger()
	service := MakeFeatureService(dispatcher, registry, repository, logger)

	cases := map[string]struct {
		FeatureName     string
		ValidationError error
		ApplyError      error
		Error           interface{}
		StatusAfter     FeatureStatus
	}{
		"success": {
			FeatureName: featureName,
			StatusAfter: FeatureStatusPending,
		},
		"unknown feature": {
			FeatureName: "notMyFeature",
			Error: UnknownFeatureError{
				FeatureName: "notMyFeature",
			},
			StatusAfter: FeatureStatusActive,
		},
		"invalid spec": {
			FeatureName:     featureName,
			ValidationError: errors.New("validation error"),
			Error:           true,
			StatusAfter:     FeatureStatusActive,
		},
		"begin apply fails": {
			FeatureName: featureName,
			ApplyError:  errors.New("failed to begin apply"),
			Error:       true,
			StatusAfter: FeatureStatusActive,
		},
	}
	spec := FeatureSpec{
		"someSpecKey": "someSpecValue",
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repository.Restore(snapshot)
			dispatcher.ApplyError = tc.ApplyError
			featureManager.ValidationError = tc.ValidationError

			err := service.Update(context.Background(), clusterID, tc.FeatureName, spec)
			switch tc.Error {
			case true:
				assert.Error(t, err)
			case nil, false:
				assert.NoError(t, err)
			default:
				assert.Equal(t, tc.Error, errors.Cause(err))
			}

			assert.Equal(t, tc.StatusAfter, repository.features[clusterID][featureName].Status)
		})
	}
}

type dummyFeatureOperationDispatcher struct {
	ApplyError      error
	DeactivateError error
}

func (d dummyFeatureOperationDispatcher) DispatchApply(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) error {
	return d.ApplyError
}

func (d dummyFeatureOperationDispatcher) DispatchDeactivate(ctx context.Context, clusterID uint, featureName string) error {
	return d.DeactivateError
}
