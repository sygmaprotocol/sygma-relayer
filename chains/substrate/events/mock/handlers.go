// Code generated by MockGen. DO NOT EDIT.
// Source: ./chains/substrate/events/handlers.go

// Package mock_events is a generated GoMock package.
package mock_events

import (
	reflect "reflect"

	message "github.com/ChainSafe/chainbridge-core/relayer/message"
	types "github.com/ChainSafe/chainbridge-core/types"
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

// MockDepositHandler is a mock of DepositHandler interface.
type MockDepositHandler struct {
	ctrl     *gomock.Controller
	recorder *MockDepositHandlerMockRecorder
}

// MockDepositHandlerMockRecorder is the mock recorder for MockDepositHandler.
type MockDepositHandlerMockRecorder struct {
	mock *MockDepositHandler
}

// NewMockDepositHandler creates a new mock instance.
func NewMockDepositHandler(ctrl *gomock.Controller) *MockDepositHandler {
	mock := &MockDepositHandler{ctrl: ctrl}
	mock.recorder = &MockDepositHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDepositHandler) EXPECT() *MockDepositHandlerMockRecorder {
	return m.recorder
}

// HandleDeposit mocks base method.
func (m *MockDepositHandler) HandleDeposit(sourceID, destID uint8, nonce uint64, resourceID types.ResourceID, calldata []byte, depositType message.TransferType, handlerResponse []byte) (*message.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleDeposit", sourceID, destID, nonce, resourceID, calldata, depositType, handlerResponse)
	ret0, _ := ret[0].(*message.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HandleDeposit indicates an expected call of HandleDeposit.
func (mr *MockDepositHandlerMockRecorder) HandleDeposit(sourceID, destID, nonce, resourceID, calldata, depositType, handlerResponse interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleDeposit", reflect.TypeOf((*MockDepositHandler)(nil).HandleDeposit), sourceID, destID, nonce, resourceID, calldata, depositType, handlerResponse)
}
