// Code generated by mockery v1.0.0. DO NOT EDIT.

package kubernetes

import context "context"
import mock "github.com/stretchr/testify/mock"

// MockDynamicFileClient is an autogenerated mock type for the DynamicFileClient type
type MockDynamicFileClient struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, file
func (_m *MockDynamicFileClient) Create(ctx context.Context, file []byte) error {
	ret := _m.Called(ctx, file)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte) error); ok {
		r0 = rf(ctx, file)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
