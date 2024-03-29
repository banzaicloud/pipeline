//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by mga tool. DO NOT EDIT.

package secrettype

import (
	"context"
	"github.com/stretchr/testify/mock"
)

// MockService is an autogenerated mock for the Service type.
type MockService struct {
	mock.Mock
}

// GetSecretType provides a mock function.
func (_m *MockService) GetSecretType(ctx context.Context, secretType string) (secretTypeDef TypeDefinition, err error) {
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

// ListSecretTypes provides a mock function.
func (_m *MockService) ListSecretTypes(ctx context.Context) (secretTypes map[string]TypeDefinition, err error) {
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
