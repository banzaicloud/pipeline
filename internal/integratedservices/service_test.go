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

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegratedServiceService_List(t *testing.T) {
	clusterID := uint(1)
	repository := NewInMemoryIntegratedServiceRepository(map[uint][]IntegratedService{
		clusterID: {
			{
				Name: "myActiveIntegratedService",
				Spec: IntegratedServiceSpec{
					"someSpecKey": "someSpecValue",
				},
				Status: IntegratedServiceStatusActive,
			},
			{
				Name: "myPendingIntegratedService",
				Spec: IntegratedServiceSpec{
					"mySpecKey": "mySpecValue",
				},
				Status: IntegratedServiceStatusPending,
			},
			{
				Name: "myErrorIntegratedService",
				Spec: IntegratedServiceSpec{
					"mySpecKey": "mySpecValue",
				},
				Status: IntegratedServiceStatusError,
			},
		},
	})
	registry := MakeIntegratedServiceManagerRegistry([]IntegratedServiceManager{
		&dummyIntegratedServiceManager{
			TheName: "myInactiveIntegratedService",
			Output: IntegratedServiceOutput{
				"someOutputKey": "someOutputValue",
			},
		},
		&dummyIntegratedServiceManager{
			TheName: "myPendingIntegratedService",
			Output: IntegratedServiceOutput{
				"someOutputKey": "someOutputValue",
			},
		},
		&dummyIntegratedServiceManager{
			TheName: "myActiveIntegratedService",
			Output: IntegratedServiceOutput{
				"someOutputKey": "someOutputValue",
			},
		},
		&dummyIntegratedServiceManager{
			TheName: "myErrorIntegratedService",
			Output: IntegratedServiceOutput{
				"someOutputKey": "someOutputValue",
			},
		},
	})
	expected := []IntegratedService{
		{
			Name:   "myActiveIntegratedService",
			Status: IntegratedServiceStatusActive,
		},
		{
			Name:   "myPendingIntegratedService",
			Status: IntegratedServiceStatusPending,
		},
		{
			Name:   "myErrorIntegratedService",
			Status: IntegratedServiceStatusError,
		},
	}
	logger := NoopLogger{}
	service := MakeIntegratedServiceService(nil, registry, repository, logger)

	integratedServices, err := service.List(context.Background(), clusterID)
	require.NoError(t, err)
	assert.ElementsMatch(t, expected, integratedServices)
}

func TestIntegratedServiceService_Details(t *testing.T) {
	clusterID := uint(1)
	registry := MakeIntegratedServiceManagerRegistry([]IntegratedServiceManager{
		&dummyIntegratedServiceManager{
			TheName: "myActiveIntegratedService",
			Output: IntegratedServiceOutput{
				"myOutputKey": "myOutputValue",
			},
		},
		&dummyIntegratedServiceManager{
			TheName: "myInactiveIntegratedService",
			Output: IntegratedServiceOutput{
				"myOutputKey": "myOutputValue",
			},
		},
	})
	repository := NewInMemoryIntegratedServiceRepository(map[uint][]IntegratedService{
		clusterID: {
			{
				Name: "myActiveIntegratedService",
				Spec: IntegratedServiceSpec{
					"mySpecKey": "mySpecValue",
				},
				Status: IntegratedServiceStatusActive,
			},
		},
	})
	logger := NoopLogger{}
	service := MakeIntegratedServiceService(nil, registry, repository, logger)

	cases := map[string]struct {
		IntegratedServiceName string
		Result                IntegratedService
		Error                 error
	}{
		"active integrated service": {
			IntegratedServiceName: "myActiveIntegratedService",
			Result: IntegratedService{
				Name: "myActiveIntegratedService",
				Spec: IntegratedServiceSpec{
					"mySpecKey": "mySpecValue",
				},
				Output: IntegratedServiceOutput{
					"myOutputKey": "myOutputValue",
				},
				Status: IntegratedServiceStatusActive,
			},
		},
		"inactive integrated service": {
			IntegratedServiceName: "myInactiveIntegratedService",
			Result: IntegratedService{
				Name:   "myInactiveIntegratedService",
				Status: IntegratedServiceStatusInactive,
			},
		},
		"unknown integrated service": {
			IntegratedServiceName: "myUnknownIntegratedService",
			Error: UnknownIntegratedServiceError{
				IntegratedServiceName: "myUnknownIntegratedService",
			},
		},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			integratedService, err := service.Details(context.Background(), clusterID, tc.IntegratedServiceName)
			switch tc.Error {
			case nil:
				require.NoError(t, err)
				assert.Equal(t, tc.Result, integratedService)
			default:
				assert.Error(t, err)
				assert.Equal(t, tc.Error, errors.Cause(err))
			}
		})
	}
}

func TestIntegratedServiceService_Activate(t *testing.T) {
	clusterID := uint(1)
	integratedServiceName := "myIntegratedService"
	dispatcher := &dummyIntegratedServiceOperationDispatcher{}
	integratedServiceManager := &dummyIntegratedServiceManager{
		TheName: integratedServiceName,
		Output: IntegratedServiceOutput{
			"someKey": "someValue",
		},
	}
	registry := MakeIntegratedServiceManagerRegistry([]IntegratedServiceManager{integratedServiceManager})
	logger := NoopLogger{}

	cases := map[string]struct {
		IntegratedServiceName  string
		ValidationError        error
		ApplyError             error
		Error                  interface{}
		IntegratedServiceSaved bool
		InitialServices        map[uint][]IntegratedService
	}{
		"success": {
			IntegratedServiceName:  integratedServiceName,
			IntegratedServiceSaved: true,
		},
		"unknown integrated service": {
			IntegratedServiceName: "notMyIntegratedService",
			Error: UnknownIntegratedServiceError{
				IntegratedServiceName: "notMyIntegratedService",
			},
		},
		"invalid spec": {
			IntegratedServiceName: integratedServiceName,
			ValidationError:       errors.New("validation error"),
			Error:                 true,
		},
		"begin apply fails": {
			IntegratedServiceName: integratedServiceName,
			ApplyError:            errors.New("failed to begin apply"),
			Error:                 true,
		},
		"already active service": {
			IntegratedServiceName: integratedServiceName,
			InitialServices: map[uint][]IntegratedService{
				clusterID: {
					{
						Name:   integratedServiceName,
						Spec:   IntegratedServiceSpec{},
						Status: IntegratedServiceStatusActive,
					},
				},
			},
			Error: serviceAlreadyActiveError{
				ServiceName: integratedServiceName,
			},
			IntegratedServiceSaved: true,
		},
	}
	spec := IntegratedServiceSpec{
		"mySpecKey": "mySpecValue",
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			repository := NewInMemoryIntegratedServiceRepository(tc.InitialServices)
			service := MakeIntegratedServiceService(dispatcher, registry, repository, logger)
			dispatcher.ApplyError = tc.ApplyError
			integratedServiceManager.ValidationError = tc.ValidationError

			err := service.Activate(context.Background(), clusterID, tc.IntegratedServiceName, spec)
			switch tc.Error {
			case true:
				assert.Error(t, err)
			case nil, false:
				assert.NoError(t, err)
			default:
				assert.Equal(t, tc.Error, errors.Cause(err))
			}

			if tc.IntegratedServiceSaved {
				assert.NotEmpty(t, repository.integratedServices[clusterID])
			} else {
				assert.Empty(t, repository.integratedServices[clusterID])
			}
		})
	}
}

func TestIntegratedServiceService_Deactivate(t *testing.T) {
	clusterID := uint(1)
	integratedServiceName := "myIntegratedService"
	dispatcher := &dummyIntegratedServiceOperationDispatcher{}
	registry := MakeIntegratedServiceManagerRegistry([]IntegratedServiceManager{
		dummyIntegratedServiceManager{
			TheName: integratedServiceName,
			Output: IntegratedServiceOutput{
				"someKey": "someValue",
			},
		},
	})
	repository := NewInMemoryIntegratedServiceRepository(map[uint][]IntegratedService{
		clusterID: {
			{
				Name: integratedServiceName,
				Spec: IntegratedServiceSpec{
					"mySpecKey": "mySpecValue",
				},
				Status: IntegratedServiceStatusActive,
			},
		},
	})
	snapshot := repository.Snapshot()
	logger := NoopLogger{}
	service := MakeIntegratedServiceService(dispatcher, registry, repository, logger)

	cases := map[string]struct {
		IntegratedServiceName string
		DeactivateError       error
		Error                 interface{}
		StatusAfter           IntegratedServiceStatus
	}{
		"success": {
			IntegratedServiceName: integratedServiceName,
			StatusAfter:           IntegratedServiceStatusPending,
		},
		"unknown integrated service": {
			IntegratedServiceName: "notMyIntegratedService",
			Error: UnknownIntegratedServiceError{
				IntegratedServiceName: "notMyIntegratedService",
			},
			StatusAfter: IntegratedServiceStatusActive,
		},
		"begin deactivate fails": {
			IntegratedServiceName: integratedServiceName,
			DeactivateError:       errors.New("failed to begin deactivate"),
			Error:                 true,
			StatusAfter:           IntegratedServiceStatusActive,
		},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			repository.Restore(snapshot)
			dispatcher.DeactivateError = tc.DeactivateError

			err := service.Deactivate(context.Background(), clusterID, tc.IntegratedServiceName)
			switch tc.Error {
			case true:
				assert.Error(t, err)
			case nil, false:
				assert.NoError(t, err)
			default:
				assert.Equal(t, tc.Error, errors.Cause(err))
			}

			assert.Equal(t, tc.StatusAfter, repository.integratedServices[clusterID][integratedServiceName].Status)
		})
	}
}

func TestIntegratedServiceService_Update(t *testing.T) {
	clusterID := uint(1)
	integratedServiceName := "myIntegratedService"
	dispatcher := &dummyIntegratedServiceOperationDispatcher{}
	integratedServiceManager := &dummyIntegratedServiceManager{
		TheName: integratedServiceName,
		Output: IntegratedServiceOutput{
			"someKey": "someValue",
		},
	}
	registry := MakeIntegratedServiceManagerRegistry([]IntegratedServiceManager{integratedServiceManager})
	repository := NewInMemoryIntegratedServiceRepository(map[uint][]IntegratedService{
		clusterID: {
			{
				Name: integratedServiceName,
				Spec: IntegratedServiceSpec{
					"mySpecKey": "mySpecValue",
				},
				Status: IntegratedServiceStatusActive,
			},
		},
	})
	snapshot := repository.Snapshot()
	logger := NoopLogger{}
	service := MakeIntegratedServiceService(dispatcher, registry, repository, logger)

	cases := map[string]struct {
		IntegratedServiceName string
		ValidationError       error
		ApplyError            error
		Error                 interface{}
		StatusAfter           IntegratedServiceStatus
	}{
		"success": {
			IntegratedServiceName: integratedServiceName,
			StatusAfter:           IntegratedServiceStatusPending,
		},
		"unknown integrated service": {
			IntegratedServiceName: "notMyIntegratedService",
			Error: UnknownIntegratedServiceError{
				IntegratedServiceName: "notMyIntegratedService",
			},
			StatusAfter: IntegratedServiceStatusActive,
		},
		"invalid spec": {
			IntegratedServiceName: integratedServiceName,
			ValidationError:       errors.New("validation error"),
			Error:                 true,
			StatusAfter:           IntegratedServiceStatusActive,
		},
		"begin apply fails": {
			IntegratedServiceName: integratedServiceName,
			ApplyError:            errors.New("failed to begin apply"),
			Error:                 true,
			StatusAfter:           IntegratedServiceStatusActive,
		},
	}
	spec := IntegratedServiceSpec{
		"someSpecKey": "someSpecValue",
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			repository.Restore(snapshot)
			dispatcher.ApplyError = tc.ApplyError
			integratedServiceManager.ValidationError = tc.ValidationError

			err := service.Update(context.Background(), clusterID, tc.IntegratedServiceName, spec)
			switch tc.Error {
			case true:
				assert.Error(t, err)
			case nil, false:
				assert.NoError(t, err)
			default:
				assert.Equal(t, tc.Error, errors.Cause(err))
			}

			assert.Equal(t, tc.StatusAfter, repository.integratedServices[clusterID][integratedServiceName].Status)
		})
	}
}

type dummyIntegratedServiceOperationDispatcher struct {
	ApplyError      error
	DeactivateError error
}

func (d dummyIntegratedServiceOperationDispatcher) DispatchApply(ctx context.Context, clusterID uint, integratedServiceName string, spec IntegratedServiceSpec) error {
	return d.ApplyError
}

func (d dummyIntegratedServiceOperationDispatcher) DispatchDeactivate(ctx context.Context, clusterID uint, integratedServiceName string, spec IntegratedServiceSpec) error {
	return d.DeactivateError
}

func (d dummyIntegratedServiceOperationDispatcher) IsBeingDispatched(ctx context.Context, clusterID uint, integratedServiceName string) (bool, error) {
	return false, nil
}
