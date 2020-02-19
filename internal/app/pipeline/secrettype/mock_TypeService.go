// Code generated by mockery v1.0.0. DO NOT EDIT.

package secrettype

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockTypeService is an autogenerated mock type for the TypeService type
type MockTypeService struct {
	mock.Mock
}

// GetSecretType provides a mock function with given fields: ctx, secretType
func (_m *MockTypeService) GetSecretType(ctx context.Context, secretType string) (TypeDefinition, error) {
	ret := _m.Called(ctx, secretType)

	var r0 TypeDefinition
	if rf, ok := ret.Get(0).(func(context.Context, string) TypeDefinition); ok {
		r0 = rf(ctx, secretType)
	} else {
		r0 = ret.Get(0).(TypeDefinition)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, secretType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListSecretTypes provides a mock function with given fields: ctx
func (_m *MockTypeService) ListSecretTypes(ctx context.Context) (map[string]TypeDefinition, error) {
	ret := _m.Called(ctx)

	var r0 map[string]TypeDefinition
	if rf, ok := ret.Get(0).(func(context.Context) map[string]TypeDefinition); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]TypeDefinition)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}