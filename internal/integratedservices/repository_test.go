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

package integratedservices

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeIntegratedServiceNotFoundError(t *testing.T) {
	assert.True(t, IsIntegratedServiceNotFoundError(integratedServiceNotFoundError{
		clusterID:             42,
		integratedServiceName: "integratedService",
	}))
}

func TestInmemoryIntegratedServiceRepository_GetIntegratedServices(t *testing.T) {
	repository := NewInMemoryIntegratedServiceRepository(nil)

	clusterID := uint(1)
	integratedService := IntegratedService{
		Name: "myIntegratedService",
		Spec: map[string]interface{}{
			"key": "value",
		},
		Status: IntegratedServiceStatusActive,
	}

	repository.integratedServices[clusterID] = map[string]IntegratedService{
		integratedService.Name: integratedService,
	}

	integratedServices, err := repository.GetIntegratedServices(context.Background(), clusterID)
	require.NoError(t, err)

	assert.Equal(t, []IntegratedService{integratedService}, integratedServices)
}

func TestInmemoryIntegratedServiceRepository_GetIntegratedService(t *testing.T) {
	repository := NewInMemoryIntegratedServiceRepository(nil)

	clusterID := uint(1)
	integratedService := IntegratedService{
		Name: "myIntegratedService",
		Spec: map[string]interface{}{
			"key": "value",
		},
		Status: IntegratedServiceStatusActive,
	}

	repository.integratedServices[clusterID] = map[string]IntegratedService{
		integratedService.Name: integratedService,
	}

	f, err := repository.GetIntegratedService(context.Background(), clusterID, integratedService.Name)
	require.NoError(t, err)

	assert.Equal(t, integratedService, f)
}

func TestInmemoryIntegratedServiceRepository_SaveIntegratedService(t *testing.T) {
	repository := NewInMemoryIntegratedServiceRepository(nil)

	clusterID := uint(1)
	integratedServiceName := "myIntegratedService"
	spec := map[string]interface{}{
		"key": "value",
	}

	expectedIntegratedService := IntegratedService{
		Name:   integratedServiceName,
		Spec:   spec,
		Status: IntegratedServiceStatusPending,
	}

	err := repository.SaveIntegratedService(context.Background(), clusterID, integratedServiceName, spec, IntegratedServiceStatusPending)
	require.NoError(t, err)

	assert.Equal(t, expectedIntegratedService, repository.integratedServices[clusterID][integratedServiceName])
}

func TestInmemoryIntegratedServiceRepository_DeleteIntegratedService(t *testing.T) {
	repository := NewInMemoryIntegratedServiceRepository(nil)

	clusterID := uint(1)
	integratedService := IntegratedService{
		Name: "myIntegratedService",
		Spec: map[string]interface{}{
			"key": "value",
		},
		Status: IntegratedServiceStatusActive,
	}

	repository.integratedServices[clusterID] = map[string]IntegratedService{
		integratedService.Name: integratedService,
	}

	err := repository.DeleteIntegratedService(context.Background(), clusterID, integratedService.Name)
	require.NoError(t, err)

	assert.NotContains(t, repository.integratedServices[clusterID], integratedService.Name)
}
