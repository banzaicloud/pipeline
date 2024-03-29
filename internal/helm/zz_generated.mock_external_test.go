//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by mga tool. DO NOT EDIT.

package helm_test

import (
	"context"
	"github.com/stretchr/testify/mock"
)

// MockOrgService is an autogenerated mock for the OrgService type.
type MockOrgService struct {
	mock.Mock
}

// GetOrgNameByOrgID provides a mock function.
func (_m *MockOrgService) GetOrgNameByOrgID(ctx context.Context, orgID uint) (_result_0 string, _result_1 error) {
	ret := _m.Called(ctx, orgID)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, uint) string); ok {
		r0 = rf(ctx, orgID)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint) error); ok {
		r1 = rf(ctx, orgID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
