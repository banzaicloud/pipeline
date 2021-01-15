// Copyright © 2020 Banzai Cloud
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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ServiceRouterSuite struct {
	suite.Suite
	clusterID uint
	ctx       context.Context
	serviceV1 MockService
	serviceV2 MockService

	logger Logger
}

// SetupTest common fixture for each test case
func (suite *ServiceRouterSuite) SetupTest() {
	suite.clusterID = 1
	suite.ctx = context.Background()
	suite.serviceV1 = MockService{}
	suite.serviceV2 = MockService{}

	suite.logger = NoopLogger{}
}

// TestServiceRouterSuite register test cases to be run
func TestServiceRouterSuite(t *testing.T) {
	suite.Run(t, new(ServiceRouterSuite))
}

// TestNoIntegratedServices both service versions return empty slices
func (suite *ServiceRouterSuite) TestList_NoIntegratedServices() {
	// Given
	// serviceV1 and serviceV2  don't return any IS services
	suite.serviceV1.On("List", suite.ctx, suite.clusterID).Return(make([]IntegratedService, 0, 0), nil)
	suite.serviceV2.On("List", suite.ctx, suite.clusterID).Return(make([]IntegratedService, 0, 0), nil)
	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	isSlice, err := routerService.List(suite.ctx, suite.clusterID)

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
	assert.Empty(suite.T(), isSlice, "the slice of integrated services should be empty")
}

func (suite *ServiceRouterSuite) TestList_MergeV1AndV2IntegratedServices() {
	// Given
	suite.serviceV1.On("List", suite.ctx, suite.clusterID).
		Return([]IntegratedService{{
			Name:   "legacy IS",
			Status: "active",
		}}, nil)

	suite.serviceV2.On("List", suite.ctx, suite.clusterID).
		Return(
			[]IntegratedService{{
				Name:   "v2 IS",
				Status: "active",
			}}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	isSlice, err := routerService.List(suite.ctx, suite.clusterID)

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
	assert.Equal(suite.T(), 2, len(isSlice), "all integrated services (legacy and v2) should be returned")
}

func (suite *ServiceRouterSuite) TestList_V2IntegratedServicesOnly() {
	// Given
	// serviceV1 and serviceV2  don't return any IS services
	suite.serviceV1.On("List", suite.ctx, suite.clusterID).
		Return(make([]IntegratedService, 0, 0), nil)

	suite.serviceV2.On("List", suite.ctx, suite.clusterID).
		Return(
			[]IntegratedService{{
				Name:   "v2 integrated service",
				Status: "active",
			}}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	isSlice, err := routerService.List(suite.ctx, suite.clusterID)

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
	assert.Equal(suite.T(), 1, len(isSlice), "the v2 IS should be returned")
}

func (suite *ServiceRouterSuite) TestList_IntegratedServiceOnBothVersions() {
	// Given
	suite.serviceV1.On("List", suite.ctx, suite.clusterID).
		Return([]IntegratedService{
			{
				Name:   "v1-is-1", // this is the  duplicate
				Status: "Pending", // note the status
			},
			{
				Name:   "v1-is-2",
				Status: "Active",
			},
		}, nil)

	suite.serviceV2.On("List", suite.ctx, suite.clusterID).
		Return(
			[]IntegratedService{
				{
					Name:   "v1-is-1", // this is the  duplicate
					Status: "Active",
				},
				{
					Name:   "v2-is-2",
					Status: "Active",
				},
			}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	isSlice, err := routerService.List(suite.ctx, suite.clusterID)

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
	assert.Equal(suite.T(), 3, len(isSlice), "the v2 IS should be returned")
	// make sure the  v1 of the duplicate is returned
	// todo check if the ordering of elements in the slice is consistent
	assert.Equal(suite.T(), "v1-is-1", isSlice[0].Name, "the is name is the expected one")
	assert.Equal(suite.T(), "Pending", isSlice[0].Status, "the is name is the expected one")
}

func (suite *ServiceRouterSuite) TestDetails_ServiceOnV2() {
	// Given
	// the IS is not found by the legacy service
	suite.serviceV1.On("Details", suite.ctx, suite.clusterID, "IS2").
		Return(IntegratedService{
			Name:   "IS2",
			Status: IntegratedServiceStatusInactive,
		}, nil)

	// serviceV2 returns the requested IS / serviceV1 doesn't get called
	suite.serviceV2.On("Details", suite.ctx, suite.clusterID, "IS2").
		Return(IntegratedService{
			Name:   "IS2",
			Status: "ACTIVE",
		}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	isDetails, err := routerService.Details(suite.ctx, suite.clusterID, "IS2")

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
	require.NotNil(suite.T(), isDetails, "router must return with details")
	require.Equal(suite.T(), "IS2", isDetails.Name) // this might be  superfluous
}

func (suite *ServiceRouterSuite) TestDetails_ServiceOnV1() {
	// Given
	// serviceV2  doesn't return the requested IS
	suite.serviceV2.On("Details", suite.ctx, suite.clusterID, "IS1").
		Return(IntegratedService{}, errors.New("service not found on v2"))

	// serviceV1  returns the requested IS
	suite.serviceV1.On("Details", suite.ctx, suite.clusterID, "IS1").
		Return(IntegratedService{
			Name:   "IS1",
			Spec:   nil,
			Output: nil,
			Status: "PENDING",
		}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	isDetails, err := routerService.Details(suite.ctx, suite.clusterID, "IS1")

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
	require.NotNil(suite.T(), isDetails, "router must not return with error")
	require.Equal(suite.T(), "IS1", isDetails.Name)
}

func (suite *ServiceRouterSuite) TestDetails_ServiceNotFound() {
	// Given
	// serviceV2  doesn't return the requested IS
	suite.serviceV2.On("Details", suite.ctx, suite.clusterID, "IS").
		Return(IntegratedService{}, integratedServiceNotFoundError{
			clusterID:             suite.clusterID,
			integratedServiceName: "IS",
		})

	// serviceV1  returns the requested IS
	suite.serviceV1.On("Details", suite.ctx, suite.clusterID, "IS").
		Return(IntegratedService{}, integratedServiceNotFoundError{
			clusterID:             suite.clusterID,
			integratedServiceName: "IS",
		})

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	isDetails, err := routerService.Details(suite.ctx, suite.clusterID, "IS")

	// Then
	require.NotNil(suite.T(), err, "router must not return with error")
	require.True(suite.T(), IsIntegratedServiceNotFoundError(err), "router must return with notfound errors")
	require.Equal(suite.T(), IntegratedService{}, isDetails, "the details must be empty")
}

func (suite *ServiceRouterSuite) TestDeactivate_ISFoundOnLegacy() {
	// Given
	ctx := context.Background()

	// the IS is found by the legacy service
	suite.serviceV1.On("Details", ctx, suite.clusterID, "IS1").
		Return(IntegratedService{
			Name:   "IS1",
			Status: "ACTIVE",
		}, nil)

	// the activation is delegated to the legacy  service
	suite.serviceV1.On("Deactivate", ctx, suite.clusterID, "IS1").Return(nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	err := routerService.Deactivate(ctx, suite.clusterID, "IS1")

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
}

func (suite *ServiceRouterSuite) TestDeactivate_NotFoundOnLegacy() {
	// Given
	ctx := context.Background()

	// the IS is NOT found by the legacy service
	suite.serviceV1.On("Details", ctx, suite.clusterID, "IS2").
		Return(IntegratedService{}, integratedServiceNotFoundError{
			clusterID:             suite.clusterID,
			integratedServiceName: "IS2",
		})

	// the activation is delegated to the new service
	suite.serviceV2.On("Deactivate", ctx, suite.clusterID, "IS2").Return(nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	err := routerService.Deactivate(ctx, suite.clusterID, "IS2")

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
}

func (suite *ServiceRouterSuite) TestUpdate_ISFoundOnLegacy() {
	// Given
	ctx := context.Background()

	// the IS is found by the legacy service
	suite.serviceV1.On("Details", ctx, suite.clusterID, "IS1").
		Return(IntegratedService{
			Name:   "IS1",
			Status: "ACTIVE",
		}, nil)

	// the activation is delegated to the legacy  service
	suite.serviceV1.On("Update", ctx, suite.clusterID, "IS1", IntegratedServiceSpec{}).Return(nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	err := routerService.Update(ctx, suite.clusterID, "IS1", IntegratedServiceSpec{})

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
}

func (suite *ServiceRouterSuite) TestUpdate_NotFoundOnLegacy() {
	// Given
	ctx := context.Background()

	// the IS is NOT found by the legacy service
	suite.serviceV1.On("Details", ctx, suite.clusterID, "IS2").
		Return(IntegratedService{}, integratedServiceNotFoundError{
			clusterID:             suite.clusterID,
			integratedServiceName: "IS2",
		})

	// the call is delegated to the new service
	suite.serviceV2.On("Update", ctx, suite.clusterID, "IS2", IntegratedServiceSpec{}).Return(nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	err := routerService.Update(ctx, suite.clusterID, "IS2", IntegratedServiceSpec{})

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
}

func (suite *ServiceRouterSuite) TestActivate_RouteToV2() {
	// Given
	ctx := context.Background()

	// the IS is NOT Active on V1
	suite.serviceV1.On("Details", ctx, suite.clusterID, "IS").
		Return(IntegratedService{
			Name:   "IS",
			Status: IntegratedServiceStatusInactive,
		}, nil)

	// the IS is NOT Active on V2
	suite.serviceV2.On("Details", ctx, suite.clusterID, "IS").
		Return(IntegratedService{
			Name:   "IS",
			Status: IntegratedServiceStatusInactive,
		}, nil)

	// Activation on V2 succeeds
	suite.serviceV2.On("Activate", suite.ctx, suite.clusterID, "IS", IntegratedServiceSpec{}).Return(nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	err := routerService.Activate(ctx, suite.clusterID, "IS", IntegratedServiceSpec{})

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
}

func (suite *ServiceRouterSuite) TestActivate_RouteToV1() {
	// Given
	ctx := context.Background()

	// the IS is NOT Active on V1
	suite.serviceV1.On("Details", ctx, suite.clusterID, "IS").
		Return(IntegratedService{
			Name:   "IS",
			Status: IntegratedServiceStatusInactive,
		}, nil)

	// the IS is NOT Active on V2
	suite.serviceV2.On("Details", ctx, suite.clusterID, "IS").
		Return(IntegratedService{
			Name:   "IS",
			Status: IntegratedServiceStatusInactive,
		}, nil)

	// IS is  not supported by the V2
	suite.serviceV2.On("Activate", suite.ctx, suite.clusterID, "IS", IntegratedServiceSpec{}).Return(UnknownIntegratedServiceError{
		IntegratedServiceName: "IS",
	})

	// Activation on V1 succeeds
	suite.serviceV1.On("Activate", suite.ctx, suite.clusterID, "IS", IntegratedServiceSpec{}).Return(nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	err := routerService.Activate(ctx, suite.clusterID, "IS", IntegratedServiceSpec{})

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
}

func (suite *ServiceRouterSuite) TestActivate_ISActivatedAlreadyOnV1() {
	// Given
	ctx := context.Background()

	// the IS is NOT found by the legacy service
	suite.serviceV1.On("Details", ctx, suite.clusterID, "IS").
		Return(IntegratedService{
			Name:   "IS",
			Status: IntegratedServiceStatusPending,
		}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	err := routerService.Activate(ctx, suite.clusterID, "IS", IntegratedServiceSpec{})
	// Then
	require.NotNil(suite.T(), err, "router must return with error")
	require.True(suite.T(), errors.As(err, &serviceAlreadyActiveError{}))
}

func (suite *ServiceRouterSuite) TestActivate_ISActivatedAlreadyOnV2() {
	// Given
	ctx := context.Background()

	// the IS is NOT found by the legacy service
	suite.serviceV1.On("Details", ctx, suite.clusterID, "IS").
		Return(IntegratedService{
			Name:   "IS",
			Status: IntegratedServiceStatusInactive,
		}, nil)

	suite.serviceV2.On("Details", ctx, suite.clusterID, "IS").
		Return(IntegratedService{
			Name:   "IS",
			Status: IntegratedServiceStatusActive,
		}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	err := routerService.Activate(ctx, suite.clusterID, "IS", IntegratedServiceSpec{})
	// Then
	require.NotNil(suite.T(), err, "router must return with error")
	require.True(suite.T(), errors.As(err, &serviceAlreadyActiveError{}))
}

func (suite *ServiceRouterSuite) TestActivate_IntermittentErrorLegacy() {
	// Given
	// the IS is NOT found by the legacy service
	suite.serviceV1.On("Details", suite.ctx, suite.clusterID, "IS2").
		Return(IntegratedService{}, errors.New("intermittent error"))

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	err := routerService.Activate(suite.ctx, suite.clusterID, "IS2", IntegratedServiceSpec{})

	// Then
	require.NotNil(suite.T(), err, "router must return with error")
}

func (suite *ServiceRouterSuite) TestActivate_ServiceNotSupportedByV2() {
	// Given
	// the IS is NOT activated on V1
	suite.serviceV1.On("Details", suite.ctx, suite.clusterID, "IS2").
		Return(IntegratedService{
			Name:   "IS2",
			Status: IntegratedServiceStatusInactive,
		}, nil)

	// the IS is NOT supported on V2
	suite.serviceV2.On("Details", suite.ctx, suite.clusterID, "IS2").
		Return(IntegratedService{}, UnknownIntegratedServiceError{
			IntegratedServiceName: "IS2",
		})

	suite.serviceV2.On("Activate", suite.ctx, suite.clusterID, "IS2", IntegratedServiceSpec{}).
		Return(UnknownIntegratedServiceError{
			IntegratedServiceName: "IS2",
		})

	suite.serviceV1.On("Activate", suite.ctx, suite.clusterID, "IS2", IntegratedServiceSpec{}).
		Return(nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2, suite.logger)

	// When
	err := routerService.Activate(suite.ctx, suite.clusterID, "IS2", IntegratedServiceSpec{})

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
}
