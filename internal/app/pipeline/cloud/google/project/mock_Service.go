// Code generated by mockery v1.0.0. DO NOT EDIT.

package project

import (
	context "context"

	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"

	mock "github.com/stretchr/testify/mock"
)

// MockService is an autogenerated mock type for the Service type
type MockService struct {
	mock.Mock
}

// ListProjects provides a mock function with given fields: ctx, secretID
func (_m *MockService) ListProjects(ctx context.Context, secretID string) ([]cloudresourcemanager.Project, error) {
	ret := _m.Called(ctx, secretID)

	var r0 []cloudresourcemanager.Project
	if rf, ok := ret.Get(0).(func(context.Context, string) []cloudresourcemanager.Project); ok {
		r0 = rf(ctx, secretID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]cloudresourcemanager.Project)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, secretID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
