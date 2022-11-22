// Code generated by MockGen. DO NOT EDIT.
// Source: ./chains/substrate/events/handlers.go

// Package mock_events is a generated GoMock package.
package mock_events

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockChainConnection is a mock of ChainConnection interface.
type MockChainConnection struct {
	ctrl     *gomock.Controller
	recorder *MockChainConnectionMockRecorder
}

// MockChainConnectionMockRecorder is the mock recorder for MockChainConnection.
type MockChainConnectionMockRecorder struct {
	mock *MockChainConnection
}

// NewMockChainConnection creates a new mock instance.
func NewMockChainConnection(ctrl *gomock.Controller) *MockChainConnection {
	mock := &MockChainConnection{ctrl: ctrl}
	mock.recorder = &MockChainConnectionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChainConnection) EXPECT() *MockChainConnectionMockRecorder {
	return m.recorder
}

// UpdateMetatdata mocks base method.
func (m *MockChainConnection) UpdateMetatdata() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateMetatdata")
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateMetatdata indicates an expected call of UpdateMetatdata.
func (mr *MockChainConnectionMockRecorder) UpdateMetatdata() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateMetatdata", reflect.TypeOf((*MockChainConnection)(nil).UpdateMetatdata))
}