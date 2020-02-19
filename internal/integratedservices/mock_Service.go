// Code generated by mockery v1.0.0. DO NOT EDIT.

package integratedservices

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockService is an autogenerated mock type for the Service type
type MockService struct {
	mock.Mock
}

// Activate provides a mock function with given fields: ctx, clusterID, serviceName, spec
func (_m *MockService) Activate(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error {
	ret := _m.Called(ctx, clusterID, serviceName, spec)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, string, map[string]interface{}) error); ok {
		r0 = rf(ctx, clusterID, serviceName, spec)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Deactivate provides a mock function with given fields: ctx, clusterID, serviceName
func (_m *MockService) Deactivate(ctx context.Context, clusterID uint, serviceName string) error {
	ret := _m.Called(ctx, clusterID, serviceName)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, string) error); ok {
		r0 = rf(ctx, clusterID, serviceName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Details provides a mock function with given fields: ctx, clusterID, serviceName
func (_m *MockService) Details(ctx context.Context, clusterID uint, serviceName string) (IntegratedService, error) {
	ret := _m.Called(ctx, clusterID, serviceName)

	var r0 IntegratedService
	if rf, ok := ret.Get(0).(func(context.Context, uint, string) IntegratedService); ok {
		r0 = rf(ctx, clusterID, serviceName)
	} else {
		r0 = ret.Get(0).(IntegratedService)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint, string) error); ok {
		r1 = rf(ctx, clusterID, serviceName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function with given fields: ctx, clusterID
func (_m *MockService) List(ctx context.Context, clusterID uint) ([]IntegratedService, error) {
	ret := _m.Called(ctx, clusterID)

	var r0 []IntegratedService
	if rf, ok := ret.Get(0).(func(context.Context, uint) []IntegratedService); ok {
		r0 = rf(ctx, clusterID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]IntegratedService)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint) error); ok {
		r1 = rf(ctx, clusterID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update provides a mock function with given fields: ctx, clusterID, serviceName, spec
func (_m *MockService) Update(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error {
	ret := _m.Called(ctx, clusterID, serviceName, spec)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, string, map[string]interface{}) error); ok {
		r0 = rf(ctx, clusterID, serviceName, spec)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
