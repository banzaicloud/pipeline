// +build !ignore_autogenerated

// Code generated by mga tool. DO NOT EDIT.

package pke

import (
	"context"
	"github.com/stretchr/testify/mock"
)

// MockService is an autogenerated mock for the Service type.
type MockService struct {
	mock.Mock
}

// ListNodePools provides a mock function.
func (_m *MockService) ListNodePools(ctx context.Context, clusterID uint) (_result_0 []NodePool, _result_1 error) {
	ret := _m.Called(ctx, clusterID)

	var r0 []NodePool
	if rf, ok := ret.Get(0).(func(context.Context, uint) []NodePool); ok {
		r0 = rf(ctx, clusterID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]NodePool)
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

// UpdateCluster provides a mock function.
func (_m *MockService) UpdateCluster(ctx context.Context, clusterID uint, clusterUpdate ClusterUpdate) (_result_0 error) {
	ret := _m.Called(ctx, clusterID, clusterUpdate)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, ClusterUpdate) error); ok {
		r0 = rf(ctx, clusterID, clusterUpdate)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateNodePool provides a mock function.
func (_m *MockService) UpdateNodePool(ctx context.Context, clusterID uint, nodePoolName string, nodePoolUpdate NodePoolUpdate) (_result_0 string, _result_1 error) {
	ret := _m.Called(ctx, clusterID, nodePoolName, nodePoolUpdate)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, uint, string, NodePoolUpdate) string); ok {
		r0 = rf(ctx, clusterID, nodePoolName, nodePoolUpdate)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint, string, NodePoolUpdate) error); ok {
		r1 = rf(ctx, clusterID, nodePoolName, nodePoolUpdate)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
