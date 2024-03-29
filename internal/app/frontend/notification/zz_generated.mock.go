//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by mga tool. DO NOT EDIT.

package notification

import (
	"context"
	"github.com/stretchr/testify/mock"
)

// MockService is an autogenerated mock for the Service type.
type MockService struct {
	mock.Mock
}

// GetNotifications provides a mock function.
func (_m *MockService) GetNotifications(ctx context.Context) (notifications Notifications, err error) {
	ret := _m.Called(ctx)

	var r0 Notifications
	if rf, ok := ret.Get(0).(func(context.Context) Notifications); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(Notifications)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
