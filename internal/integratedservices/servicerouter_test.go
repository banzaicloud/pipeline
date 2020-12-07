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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ServiceRouterSuite struct {
	suite.Suite
	serviceV1 MockService
	serviceV2 MockService
}

// SetupTest common fixture for each test case
func (suite *ServiceRouterSuite) SetupTest() {
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
	const clusterID uint = 1
	ctx := context.Background()

	// serviceV1 and serviceV2  don't return any IS services
	suite.serviceV1.On("List", ctx, clusterID).Return(make([]IntegratedService, 0, 0), nil)
	suite.serviceV2.On("List", ctx, clusterID).Return(make([]IntegratedService, 0, 0), nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2)
	// When

	isSlice, err := routerService.List(ctx, clusterID)
	require.Nil(suite.T(), err, "router must not return with error")

	// Then
	// the resulted list must be empty
	assert.Empty(suite.T(), isSlice, "the slice of integrated services should be empty")
}

// TestNoIntegratedServices both service versions return empty slices
func (suite *ServiceRouterSuite) TestList_MergeV1AndV2IntegratedServices() {
	// Given
	const clusterID uint = 1
	ctx := context.Background()

	suite.serviceV1.On("List", ctx, clusterID).
		Return([]IntegratedService{{
			Name:   "v1 integrated service",
			Status: "active",
		}}, nil)

	suite.serviceV2.On("List", ctx, clusterID).
		Return(
			[]IntegratedService{{
				Name:   "v2 integrated service",
				Status: "active",
			}}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2)
	// When

	isSlice, err := routerService.List(ctx, clusterID)
	require.Nil(suite.T(), err, "router must not return with error")

	// Then
	// the resulted list must be empty
	assert.Equal(suite.T(), 2, len(isSlice), "all integrated service v1 and v2 should be returned")
}

// TestNoIntegratedServices both service versions return empty slices
func (suite *ServiceRouterSuite) TestList_V2IntegratedServicesOnly() {
	// Givens
	const clusterID uint = 1
	ctx := context.Background()

	// serviceV1 and serviceV2  don't return any IS services
	suite.serviceV1.On("List", ctx, clusterID).
		Return(make([]IntegratedService, 0, 0), nil)

	suite.serviceV2.On("List", ctx, clusterID).
		Return(
			[]IntegratedService{{
				Name:   "v2 integrated service",
				Status: "active",
			}}, nil)

	routerService := NewServiceRouter(&suite.serviceV1, &suite.serviceV2)
	// When

	isSlice, err := routerService.List(ctx, clusterID)
	require.Nil(suite.T(), err, "router must not return with error")

	// Then
	// the resulted list must be empty
	assert.Equal(suite.T(), 1, len(isSlice), "all integrated service v1 and v2 should be returned")
}
