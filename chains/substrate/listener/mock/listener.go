// Code generated by MockGen. DO NOT EDIT.
// Source: ./chains/substrate/listener/listener.go

// Package mock_listener is a generated GoMock package.
package mock_listener

import (
	reflect "reflect"

	message "github.com/ChainSafe/chainbridge-core/relayer/message"
	events "github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	gomock "github.com/golang/mock/gomock"
)

// MockEventHandler is a mock of EventHandler interface.
type MockEventHandler struct {
	ctrl     *gomock.Controller
	recorder *MockEventHandlerMockRecorder
}

// MockEventHandlerMockRecorder is the mock recorder for MockEventHandler.
type MockEventHandlerMockRecorder struct {
	mock *MockEventHandler
}

// NewMockEventHandler creates a new mock instance.
func NewMockEventHandler(ctrl *gomock.Controller) *MockEventHandler {
	mock := &MockEventHandler{ctrl: ctrl}
	mock.recorder = &MockEventHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventHandler) EXPECT() *MockEventHandlerMockRecorder {
	return m.recorder
}

// HandleEvents mocks base method.
func (m *MockEventHandler) HandleEvents(evts []*events.Events, msgChan chan []*message.Message) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleEvents", evts, msgChan)
	ret0, _ := ret[0].(error)
	return ret0
}

// HandleEvents indicates an expected call of HandleEvents.
func (mr *MockEventHandlerMockRecorder) HandleEvents(evts, msgChan interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleEvents", reflect.TypeOf((*MockEventHandler)(nil).HandleEvents), evts, msgChan)
}

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

// GetBlockEvents mocks base method.
func (m *MockChainConnection) GetBlockEvents(hash types.Hash) (*events.Events, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockEvents", hash)
	ret0, _ := ret[0].(*events.Events)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockEvents indicates an expected call of GetBlockEvents.
func (mr *MockChainConnectionMockRecorder) GetBlockEvents(hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockEvents", reflect.TypeOf((*MockChainConnection)(nil).GetBlockEvents), hash)
}

// GetBlockHash mocks base method.
func (m *MockChainConnection) GetBlockHash(blockNumber uint64) (types.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockHash", blockNumber)
	ret0, _ := ret[0].(types.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockHash indicates an expected call of GetBlockHash.
func (mr *MockChainConnectionMockRecorder) GetBlockHash(blockNumber interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockHash", reflect.TypeOf((*MockChainConnection)(nil).GetBlockHash), blockNumber)
}

// GetBlockLatest mocks base method.
func (m *MockChainConnection) GetBlockLatest() (*types.SignedBlock, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockLatest")
	ret0, _ := ret[0].(*types.SignedBlock)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockLatest indicates an expected call of GetBlockLatest.
func (mr *MockChainConnectionMockRecorder) GetBlockLatest() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockLatest", reflect.TypeOf((*MockChainConnection)(nil).GetBlockLatest))
}

// GetHeaderLatest mocks base method.
func (m *MockChainConnection) GetHeaderLatest() (*types.Header, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHeaderLatest")
	ret0, _ := ret[0].(*types.Header)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHeaderLatest indicates an expected call of GetHeaderLatest.
func (mr *MockChainConnectionMockRecorder) GetHeaderLatest() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHeaderLatest", reflect.TypeOf((*MockChainConnection)(nil).GetHeaderLatest))
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
