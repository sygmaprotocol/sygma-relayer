// Code generated by MockGen. DO NOT EDIT.
// Source: ./chains/substrate/listener/event-handlers.go

// Package mock_listener is a generated GoMock package.
package mock_listener

import (
	reflect "reflect"

	types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	gomock "github.com/golang/mock/gomock"
	message "github.com/sygmaprotocol/sygma-core/relayer/message"
)

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
func (m *MockDepositHandler) HandleDeposit(sourceID uint8, destID types.U8, nonce types.U64, resourceID types.Bytes32, calldata []byte, transferType types.U8) (*message.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleDeposit", sourceID, destID, nonce, resourceID, calldata, transferType)
	ret0, _ := ret[0].(*message.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HandleDeposit indicates an expected call of HandleDeposit.
func (mr *MockDepositHandlerMockRecorder) HandleDeposit(sourceID, destID, nonce, resourceID, calldata, transferType interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleDeposit", reflect.TypeOf((*MockDepositHandler)(nil).HandleDeposit), sourceID, destID, nonce, resourceID, calldata, transferType)
}
