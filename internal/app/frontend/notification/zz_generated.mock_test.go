//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by mga tool. DO NOT EDIT.

package notification

import (
	"context"
	"github.com/stretchr/testify/mock"
)

// MockStore is an autogenerated mock for the Store type.
type MockStore struct {
	mock.Mock
}

// GetActiveNotifications provides a mock function.
func (_m *MockStore) GetActiveNotifications(ctx context.Context) (_result_0 []Notification, _result_1 error) {
	ret := _m.Called(ctx)

	var r0 []Notification
	if rf, ok := ret.Get(0).(func(context.Context) []Notification); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]Notification)
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
