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

package clusterfeaturedriver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

func TestMakeEndpoints_List(t *testing.T) {
	service := new(clusterfeature.MockService)

	ctx := context.Background()
	clusterID := uint(1)

	clusterFeatureList := []clusterfeature.Feature{
		{
			Name: "example",
			Spec: map[string]interface{}{
				"hello": "world",
			},
			Output: map[string]interface{}{
				"hello": "world",
			},
			Status: "ACTIVE",
		},
	}

	service.On("List", ctx, clusterID).Return(clusterFeatureList, nil)

	e := MakeEndpoints(service).List

	req := ListClusterFeaturesRequest{
		ClusterID: clusterID,
	}

	result, err := e(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, map[string]pipeline.ClusterFeatureDetails{
		"example": {
			Spec: map[string]interface{}{
				"hello": "world",
			},
			Output: map[string]interface{}{
				"hello": "world",
			},
			Status: "ACTIVE",
		},
	}, result)

	service.AssertExpectations(t)
}

func TestMakeEndpoints_Details(t *testing.T) {
	service := new(clusterfeature.MockService)

	ctx := context.Background()
	clusterID := uint(1)
	featureName := "example"

	clusterFeatureDetails := clusterfeature.Feature{
		Name: "example",
		Spec: map[string]interface{}{
			"hello": "world",
		},
		Output: map[string]interface{}{
			"hello": "world",
		},
		Status: "ACTIVE",
	}

	service.On("Details", ctx, clusterID, featureName).Return(clusterFeatureDetails, nil)

	e := MakeEndpoints(service).Details

	req := ClusterFeatureDetailsRequest{
		ClusterID:   clusterID,
		FeatureName: featureName,
	}

	result, err := e(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, pipeline.ClusterFeatureDetails{
		Spec: map[string]interface{}{
			"hello": "world",
		},
		Output: map[string]interface{}{
			"hello": "world",
		},
		Status: "ACTIVE",
	}, result)

	service.AssertExpectations(t)
}

func TestMakeEndpoints_Activate(t *testing.T) {
	service := new(clusterfeature.MockService)

	ctx := context.Background()
	clusterID := uint(1)
	featureName := "example"
	spec := map[string]interface{}{
		"hello": "world",
	}

	service.On("Activate", ctx, clusterID, featureName, spec).Return(nil)

	e := MakeEndpoints(service).Activate

	req := ActivateClusterFeatureRequest{
		ClusterID:   clusterID,
		FeatureName: featureName,
		Spec:        spec,
	}

	result, err := e(ctx, req)

	require.NoError(t, err)
	assert.Nil(t, result)

	service.AssertExpectations(t)
}

func TestMakeEndpoints_Deactivate(t *testing.T) {
	mockService := new(clusterfeature.MockService)

	ctx := context.Background()
	clusterID := uint(1)
	featureName := "example"

	mockService.On("Deactivate", ctx, clusterID, featureName).Return(nil)

	e := MakeEndpoints(mockService).Deactivate

	req := DeactivateClusterFeatureRequest{
		ClusterID:   clusterID,
		FeatureName: featureName,
	}

	result, err := e(ctx, req)

	require.NoError(t, err)
	assert.Nil(t, result)

	mockService.AssertExpectations(t)
}

func TestMakeEndpoints_Update(t *testing.T) {
	service := new(clusterfeature.MockService)

	ctx := context.Background()
	clusterID := uint(1)
	featureName := "example"
	spec := map[string]interface{}{
		"hello": "world",
	}

	service.On("Update", ctx, clusterID, featureName, spec).Return(nil)

	e := MakeEndpoints(service).Update

	req := UpdateClusterFeatureRequest{
		ClusterID:   clusterID,
		FeatureName: featureName,
		Spec:        spec,
	}

	result, err := e(ctx, req)

	require.NoError(t, err)
	assert.Nil(t, result)

	service.AssertExpectations(t)
}
