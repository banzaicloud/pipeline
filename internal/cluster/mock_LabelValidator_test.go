// Code generated by mockery v1.0.0. DO NOT EDIT.

package cluster

import mock "github.com/stretchr/testify/mock"

// MockLabelValidator is an autogenerated mock type for the LabelValidator type
type MockLabelValidator struct {
	mock.Mock
}

// ValidateKey provides a mock function with given fields: key
func (_m *MockLabelValidator) ValidateKey(key string) error {
	ret := _m.Called(key)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ValidateValue provides a mock function with given fields: value
func (_m *MockLabelValidator) ValidateValue(value string) error {
	ret := _m.Called(value)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
