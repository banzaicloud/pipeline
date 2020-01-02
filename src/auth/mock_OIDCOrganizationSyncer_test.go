// Code generated by mockery v1.0.0. DO NOT EDIT.

package auth

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockOIDCOrganizationSyncer is an autogenerated mock type for the OIDCOrganizationSyncer type
type MockOIDCOrganizationSyncer struct {
	mock.Mock
}

// SyncOrganizations provides a mock function with given fields: ctx, user, idTokenClaims
func (_m *MockOIDCOrganizationSyncer) SyncOrganizations(ctx context.Context, user User, idTokenClaims *IDTokenClaims) error {
	ret := _m.Called(ctx, user, idTokenClaims)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, User, *IDTokenClaims) error); ok {
		r0 = rf(ctx, user, idTokenClaims)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
