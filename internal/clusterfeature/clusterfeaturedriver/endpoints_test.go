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

	"github.com/banzaicloud/pipeline/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

//go:generate sh -c "test -x ${MOCKERY} && ${MOCKERY} -name FeatureService -inpkg -testonly"

func TestMakeListEndpoint(t *testing.T) {
	featureService := &MockFeatureService{}

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

	featureService.On("List", ctx, clusterID).Return(clusterFeatureList, nil)

	e := MakeListEndpoint(featureService)

	req := ListClusterFeaturesRequest{
		ClusterID: clusterID,
	}

	result, err := e(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, map[string]client.ClusterFeatureDetails{
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

	featureService.AssertExpectations(t)
}

func TestMakeDetailsEndpoint(t *testing.T) {
	featureService := &MockFeatureService{}

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

	featureService.On("Details", ctx, clusterID, featureName).Return(clusterFeatureDetails, nil)

	e := MakeDetailsEndpoint(featureService)

	req := ClusterFeatureDetailsRequest{
		ClusterID:   clusterID,
		FeatureName: featureName,
	}

	result, err := e(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, client.ClusterFeatureDetails{
		Spec: map[string]interface{}{
			"hello": "world",
		},
		Output: map[string]interface{}{
			"hello": "world",
		},
		Status: "ACTIVE",
	}, result)

	featureService.AssertExpectations(t)
}

func TestMakeActivateEndpoint(t *testing.T) {
	featureService := &MockFeatureService{}

	ctx := context.Background()
	clusterID := uint(1)
	featureName := "example"
	spec := map[string]interface{}{
		"hello": "world",
	}

	featureService.On("Activate", ctx, clusterID, featureName, spec).Return(nil)

	e := MakeActivateEndpoint(featureService)

	req := ActivateClusterFeatureRequest{
		ClusterID:   clusterID,
		FeatureName: featureName,
		Spec:        spec,
	}

	result, err := e(ctx, req)

	require.NoError(t, err)
	assert.Nil(t, result)

	featureService.AssertExpectations(t)
}

func TestMakeDeactivateEndpoint(t *testing.T) {
	featureService := &MockFeatureService{}

	ctx := context.Background()
	clusterID := uint(1)
	featureName := "example"

	featureService.On("Deactivate", ctx, clusterID, featureName).Return(nil)

	e := MakeDeactivateEndpoint(featureService)

	req := DeactivateClusterFeatureRequest{
		ClusterID:   clusterID,
		FeatureName: featureName,
	}

	result, err := e(ctx, req)

	require.NoError(t, err)
	assert.Nil(t, result)

	featureService.AssertExpectations(t)
}

func TestMakeUpdateEndpoint(t *testing.T) {
	featureService := &MockFeatureService{}

	ctx := context.Background()
	clusterID := uint(1)
	featureName := "example"
	spec := map[string]interface{}{
		"hello": "world",
	}

	featureService.On("Update", ctx, clusterID, featureName, spec).Return(nil)

	e := MakeUpdateEndpoint(featureService)

	req := UpdateClusterFeatureRequest{
		ClusterID:   clusterID,
		FeatureName: featureName,
		Spec:        spec,
	}

	result, err := e(ctx, req)

	require.NoError(t, err)
	assert.Nil(t, result)

	featureService.AssertExpectations(t)
}
