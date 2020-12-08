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
	serviceV1 MockService
	serviceV2 MockService
}

// SetupTest common fixture for each test case
func (suite *ServiceRouterSuite) SetupTest() {
	suite.clusterID = 1
	suite.serviceV1 = MockService{}
	suite.serviceV2 = MockService{}
}

// TestServiceRouterSuite register test cases to be run
func TestServiceRouterSuite(t *testing.T) {
	suite.Run(t, new(ServiceRouterSuite))
}

// TestNoIntegratedServices both service versions return empty slices
func (suite *ServiceRouterSuite) TestList_NoIntegratedServices() {
	// Given
	ctx := context.Background()

	// serviceV1 and serviceV2  don't return any IS services
	suite.serviceV1.On("List", ctx, suite.clusterID).Return(make([]IntegratedService, 0, 0), nil)
	suite.serviceV2.On("List", ctx, suite.clusterID).Return(make([]IntegratedService, 0, 0), nil)
	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2)

	// When
	isSlice, err := routerService.List(ctx, suite.clusterID)

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
	assert.Empty(suite.T(), isSlice, "the slice of integrated services should be empty")
}

func (suite *ServiceRouterSuite) TestList_MergeV1AndV2IntegratedServices() {
	// Given
	ctx := context.Background()

	suite.serviceV1.On("List", ctx, suite.clusterID).
		Return([]IntegratedService{{
			Name:   "v1 integrated service",
			Status: "active",
		}}, nil)

	suite.serviceV2.On("List", ctx, suite.clusterID).
		Return(
			[]IntegratedService{{
				Name:   "v2 integrated service",
				Status: "active",
			}}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2)

	// When
	isSlice, err := routerService.List(ctx, suite.clusterID)

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
	assert.Equal(suite.T(), 2, len(isSlice), "all integrated services (v1 and v2) should be returned")
}

func (suite *ServiceRouterSuite) TestList_V2IntegratedServicesOnly() {
	// Given
	ctx := context.Background()

	// serviceV1 and serviceV2  don't return any IS services
	suite.serviceV1.On("List", ctx, suite.clusterID).
		Return(make([]IntegratedService, 0, 0), nil)

	suite.serviceV2.On("List", ctx, suite.clusterID).
		Return(
			[]IntegratedService{{
				Name:   "v2 integrated service",
				Status: "active",
			}}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2)

	// When
	isSlice, err := routerService.List(ctx, suite.clusterID)

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
	assert.Equal(suite.T(), 1, len(isSlice), "the v2 IS should be returned")
}

func (suite *ServiceRouterSuite) TestDetails_ServiceOnV2() {
	// Given
	ctx := context.Background()

	// serviceV2 returns the requested IS / serviceV1 doesn't get called
	suite.serviceV2.On("Details", ctx, suite.clusterID, "IS2").
		Return(IntegratedService{
			Name:   "IS2",
			Status: "ACTIVE",
		}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2)

	// When
	isDetails, err := routerService.Details(ctx, suite.clusterID, "IS2")

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
	require.NotNil(suite.T(), isDetails, "router must return with details")
	require.Equal(suite.T(), "IS2", isDetails.Name) // this might be  superfluous
}

func (suite *ServiceRouterSuite) TestDetails_ServiceOnV1() {
	// Given
	ctx := context.Background()

	// serviceV2  doesn't return the requested IS
	suite.serviceV2.On("Details", ctx, suite.clusterID, "IS1").
		Return(IntegratedService{}, errors.New("service not found on v2"))

	// serviceV1  returns the requested IS
	suite.serviceV1.On("Details", ctx, suite.clusterID, "IS1").
		Return(IntegratedService{
			Name:   "IS1",
			Spec:   nil,
			Output: nil,
			Status: "PENDING",
		}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2)

	// When
	isDetails, err := routerService.Details(ctx, suite.clusterID, "IS1")

	// Then
	require.Nil(suite.T(), err, "router must not return with error")
	require.NotNil(suite.T(), isDetails, "router must not return with error")
	require.Equal(suite.T(), "IS1", isDetails.Name)
}

func (suite *ServiceRouterSuite) TestDetails_ServiceNotFound() {
	// Given
	ctx := context.Background()

	// serviceV2  doesn't return the requested IS
	suite.serviceV2.On("Details", ctx, suite.clusterID, "IS").
		Return(IntegratedService{}, integratedServiceNotFoundError{
			clusterID:             suite.clusterID,
			integratedServiceName: "IS",
		})

	// serviceV1  returns the requested IS
	suite.serviceV1.On("Details", ctx, suite.clusterID, "IS").
		Return(IntegratedService{}, integratedServiceNotFoundError{
			clusterID:             suite.clusterID,
			integratedServiceName: "IS",
		})

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2)

	// When
	isDetails, err := routerService.Details(ctx, suite.clusterID, "IS")

	// Then
	require.NotNil(suite.T(), err, "router must not return with error")
	require.True(suite.T(), IsIntegratedServiceNotFoundError(err), "router must return with notfound errors")
	require.Equal(suite.T(), IntegratedService{}, isDetails, "the details must be empty")

}
