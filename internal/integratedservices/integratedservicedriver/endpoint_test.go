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

package integratedservicedriver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

func TestMakeEndpoints_List(t *testing.T) {
	service := new(integratedservices.MockService)

	clusterID := uint(1)

	integratedServiceList := []integratedservices.IntegratedService{
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

	service.On("List", mock.Anything, clusterID).Return(integratedServiceList, nil)

	e := MakeEndpoints(service).List

	req := ListIntegratedServicesRequest{
		ClusterID: clusterID,
	}

	result, err := e(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, map[string]pipeline.IntegratedServiceDetails{
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
	service := new(integratedservices.MockService)

	clusterID := uint(1)
	integratedServiceName := "example"

	integratedServiceDetails := integratedservices.IntegratedService{
		Name: "example",
		Spec: map[string]interface{}{
			"hello": "world",
		},
		Output: map[string]interface{}{
			"hello": "world",
		},
		Status: "ACTIVE",
	}

	service.On("Details", mock.Anything, clusterID, integratedServiceName).Return(integratedServiceDetails, nil)

	e := MakeEndpoints(service).Details

	req := IntegratedServiceDetailsRequest{
		ClusterID:             clusterID,
		IntegratedServiceName: integratedServiceName,
	}

	result, err := e(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, pipeline.IntegratedServiceDetails{
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
	service := new(integratedservices.MockService)

	clusterID := uint(1)
	integratedServiceName := "example"
	spec := map[string]interface{}{
		"hello": "world",
	}

	service.On("Activate", mock.Anything, clusterID, integratedServiceName, spec).Return(nil)

	e := MakeEndpoints(service).Activate

	req := ActivateIntegratedServiceRequest{
		ClusterID:             clusterID,
		IntegratedServiceName: integratedServiceName,
		Spec:                  spec,
	}

	result, err := e(context.Background(), req)

	require.NoError(t, err)
	assert.Nil(t, result)

	service.AssertExpectations(t)
}

func TestMakeEndpoints_Deactivate(t *testing.T) {
	mockService := new(integratedservices.MockService)

	clusterID := uint(1)
	integratedServiceName := "example"

	mockService.On("Deactivate", mock.Anything, clusterID, integratedServiceName).Return(nil)

	e := MakeEndpoints(mockService).Deactivate

	req := DeactivateIntegratedServiceRequest{
		ClusterID:             clusterID,
		IntegratedServiceName: integratedServiceName,
	}

	result, err := e(context.Background(), req)

	require.NoError(t, err)
	assert.Nil(t, result)

	mockService.AssertExpectations(t)
}

func TestMakeEndpoints_Update(t *testing.T) {
	service := new(integratedservices.MockService)

	clusterID := uint(1)
	integratedServiceName := "example"
	spec := map[string]interface{}{
		"hello": "world",
	}

	service.On("Update", mock.Anything, clusterID, integratedServiceName, spec).Return(nil)

	e := MakeEndpoints(service).Update

	req := UpdateIntegratedServiceRequest{
		ClusterID:             clusterID,
		IntegratedServiceName: integratedServiceName,
		Spec:                  spec,
	}

	result, err := e(context.Background(), req)

	require.NoError(t, err)
	assert.Nil(t, result)

	service.AssertExpectations(t)
}
