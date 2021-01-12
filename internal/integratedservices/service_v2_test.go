// Copyright © 2021 Banzai Cloud
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

	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/common"
)

func TestISServiceV2_List(t *testing.T) {
	ctx := context.TODO()
	clusterID := uint(123)

	type fields struct {
		registry   IntegratedServiceManagerRegistry
		dispatcher IntegratedServiceOperationDispatcher
		repository IntegratedServiceRepository
		logger     common.Logger
	}
	type args struct {
		ctx       context.Context
		clusterID uint
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		setupMocks func(
			registry *IntegratedServiceManagerRegistry,
			dispatcher *IntegratedServiceOperationDispatcher,
			repository *IntegratedServiceRepository,
			arguments args,
		)
		expect func(*testing.T, []IntegratedService, error)
	}{
		{
			name: "list inactive service with running workflow will be pending",
			fields: fields{
				registry:   &MockIntegratedServiceManagerRegistry{},
				dispatcher: &MockIntegratedServiceOperationDispatcher{},
				repository: &MockIntegratedServiceRepository{},
				logger:     common.NoopLogger{},
			},
			args: args{
				ctx:       ctx,
				clusterID: clusterID,
			},
			setupMocks: func(
				registry *IntegratedServiceManagerRegistry,
				dispatcher *IntegratedServiceOperationDispatcher,
				repository *IntegratedServiceRepository,
				arguments args) {
				(*repository).(*MockIntegratedServiceRepository).On("GetIntegratedServices", ctx, clusterID).Return(nil, nil)
				(*registry).(*MockIntegratedServiceManagerRegistry).On("GetIntegratedServiceNames").Return([]string{"fake"})
				(*dispatcher).(*MockIntegratedServiceOperationDispatcher).On("IsBeingDispatched", ctx, clusterID, "fake").Return(true, nil)
			},
			expect: func(t *testing.T, services []IntegratedService, err error) {
				require.NoError(t, err)
				require.Contains(t, services, IntegratedService{
					Name:   "fake",
					Status: IntegratedServiceStatusPending,
				})
			},
		},
		{
			name: "list active service with no running workflow will be active",
			fields: fields{
				registry:   &MockIntegratedServiceManagerRegistry{},
				dispatcher: &MockIntegratedServiceOperationDispatcher{},
				repository: &MockIntegratedServiceRepository{},
				logger:     common.NoopLogger{},
			},
			args: args{
				ctx:       ctx,
				clusterID: clusterID,
			},
			setupMocks: func(
				registry *IntegratedServiceManagerRegistry,
				dispatcher *IntegratedServiceOperationDispatcher,
				repository *IntegratedServiceRepository,
				arguments args) {
				(*repository).(*MockIntegratedServiceRepository).On("GetIntegratedServices", ctx, clusterID).
					Return([]IntegratedService{
						{
							Name:   "fake",
							Status: IntegratedServiceStatusActive,
						},
					}, nil)
				(*registry).(*MockIntegratedServiceManagerRegistry).On("GetIntegratedServiceNames").
					Return([]string{"fake"})
				(*dispatcher).(*MockIntegratedServiceOperationDispatcher).On("IsBeingDispatched", ctx, clusterID, "fake").
					Return(false, nil)
			},
			expect: func(t *testing.T, services []IntegratedService, err error) {
				require.NoError(t, err)
				require.Contains(t, services, IntegratedService{
					Name:   "fake",
					Status: IntegratedServiceStatusActive,
				})
			},
		},
		{
			name: "list won't return unsupported services",
			fields: fields{
				registry:   &MockIntegratedServiceManagerRegistry{},
				dispatcher: &MockIntegratedServiceOperationDispatcher{},
				repository: &MockIntegratedServiceRepository{},
				logger:     common.NoopLogger{},
			},
			args: args{
				ctx:       ctx,
				clusterID: clusterID,
			},
			setupMocks: func(
				registry *IntegratedServiceManagerRegistry,
				dispatcher *IntegratedServiceOperationDispatcher,
				repository *IntegratedServiceRepository,
				arguments args) {
				(*repository).(*MockIntegratedServiceRepository).On("GetIntegratedServices", ctx, clusterID).
					Return([]IntegratedService{
						{
							Name:   "fake",
							Status: IntegratedServiceStatusActive,
						},
					}, nil)
				(*registry).(*MockIntegratedServiceManagerRegistry).On("GetIntegratedServiceNames").
					// no integrated service supported in config
					Return(nil)
				(*dispatcher).(*MockIntegratedServiceOperationDispatcher).On("IsBeingDispatched", ctx, clusterID, "fake").
					Return(false, nil)
			},
			expect: func(t *testing.T, services []IntegratedService, err error) {
				require.NoError(t, err)
				require.NotContains(t, services, IntegratedService{
					Name:   "fake",
					Status: IntegratedServiceStatusActive,
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks(&tt.fields.registry, &tt.fields.dispatcher, &tt.fields.repository, tt.args)
			service := NewISServiceV2(tt.fields.registry, tt.fields.dispatcher, tt.fields.repository, tt.fields.logger)

			services, err := service.List(tt.args.ctx, tt.args.clusterID)
			tt.expect(t, services, err)
		})
	}
}

func TestISServiceV2_Details(t *testing.T) {
	ctx := context.TODO()
	clusterID := uint(123)

	type fields struct {
		registry   IntegratedServiceManagerRegistry
		dispatcher IntegratedServiceOperationDispatcher
		repository IntegratedServiceRepository
		logger     common.Logger
	}
	type args struct {
		ctx         context.Context
		clusterID   uint
		serviceName string
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		setupMocks func(
			registry *IntegratedServiceManagerRegistry,
			dispatcher *IntegratedServiceOperationDispatcher,
			repository *IntegratedServiceRepository,
			arguments args,
		)
		expect func(*testing.T, IntegratedService, error)
	}{
		{
			name: "details returns service as pending if dispatched",
			fields: fields{
				registry:   &MockIntegratedServiceManagerRegistry{},
				dispatcher: &MockIntegratedServiceOperationDispatcher{},
				repository: &MockIntegratedServiceRepository{},
				logger:     common.NoopLogger{},
			},
			args: args{
				ctx:         ctx,
				clusterID:   clusterID,
				serviceName: "fake",
			},
			setupMocks: func(
				registry *IntegratedServiceManagerRegistry,
				dispatcher *IntegratedServiceOperationDispatcher,
				repository *IntegratedServiceRepository,
				arguments args) {
				(*dispatcher).(*MockIntegratedServiceOperationDispatcher).On("IsBeingDispatched", arguments.ctx, arguments.clusterID, arguments.serviceName).
					Return(true, nil)
			},
			expect: func(t *testing.T, service IntegratedService, err error) {
				require.NoError(t, err)
				require.Equal(t, service, IntegratedService{
					Name:   "fake",
					Status: IntegratedServiceStatusPending,
				})
			},
		},
		{
			name: "details returns service as is if not dispatched",
			fields: fields{
				registry:   &MockIntegratedServiceManagerRegistry{},
				dispatcher: &MockIntegratedServiceOperationDispatcher{},
				repository: &MockIntegratedServiceRepository{},
				logger:     common.NoopLogger{},
			},
			args: args{
				ctx:         ctx,
				clusterID:   clusterID,
				serviceName: "fake",
			},
			setupMocks: func(
				registry *IntegratedServiceManagerRegistry,
				dispatcher *IntegratedServiceOperationDispatcher,
				repository *IntegratedServiceRepository,
				arguments args) {
				(*repository).(*MockIntegratedServiceRepository).On("GetIntegratedService", arguments.ctx, arguments.clusterID, arguments.serviceName).
					Return(IntegratedService{
						Name:   "fake",
						Status: IntegratedServiceStatusActive,
						Spec:   IntegratedServiceSpec{"spec": "spec1"},
						Output: IntegratedServiceOutput{"out": "out1"},
					}, nil)
				(*dispatcher).(*MockIntegratedServiceOperationDispatcher).On("IsBeingDispatched", arguments.ctx, arguments.clusterID, arguments.serviceName).
					Return(false, nil)
			},
			expect: func(t *testing.T, service IntegratedService, err error) {
				require.NoError(t, err)
				require.Equal(t, service, IntegratedService{
					Name:   "fake",
					Status: IntegratedServiceStatusActive,
					Spec:   IntegratedServiceSpec{"spec": "spec1"},
					Output: IntegratedServiceOutput{"out": "out1"},
				})
			},
		},
		{
			name: "details returns service as inactive if it doesnt exist and not dispatched",
			fields: fields{
				registry:   &MockIntegratedServiceManagerRegistry{},
				dispatcher: &MockIntegratedServiceOperationDispatcher{},
				repository: &MockIntegratedServiceRepository{},
				logger:     common.NoopLogger{},
			},
			args: args{
				ctx:         ctx,
				clusterID:   clusterID,
				serviceName: "fake",
			},
			setupMocks: func(
				registry *IntegratedServiceManagerRegistry,
				dispatcher *IntegratedServiceOperationDispatcher,
				repository *IntegratedServiceRepository,
				arguments args) {
				(*repository).(*MockIntegratedServiceRepository).On("GetIntegratedService", arguments.ctx, arguments.clusterID, arguments.serviceName).
					Return(IntegratedService{}, integratedServiceNotFoundError{})
				(*dispatcher).(*MockIntegratedServiceOperationDispatcher).On("IsBeingDispatched", arguments.ctx, arguments.clusterID, arguments.serviceName).
					Return(false, nil)
			},
			expect: func(t *testing.T, service IntegratedService, err error) {
				require.NoError(t, err)
				require.Equal(t, service, IntegratedService{
					Name:   "fake",
					Status: IntegratedServiceStatusInactive,
				})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := ISServiceV2{
				managerRegistry: tt.fields.registry,
				dispatcher:      tt.fields.dispatcher,
				repository:      tt.fields.repository,
				logger:          tt.fields.logger,
			}
			tt.setupMocks(&tt.fields.registry, &tt.fields.dispatcher, &tt.fields.repository, tt.args)
			got, err := i.Details(tt.args.ctx, tt.args.clusterID, tt.args.serviceName)
			tt.expect(t, got, err)
		})
	}
}
