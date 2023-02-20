// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package listener_test

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/listener"
	mock_events "github.com/ChainSafe/sygma-relayer/chains/substrate/listener/mock"

	"testing"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type SystemUpdateHandlerTestSuite struct {
	suite.Suite
	conn                *mock_events.MockChainConnection
	systemUpdateHandler *listener.SystemUpdateEventHandler
}

func TestRunSystemUpdateHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SystemUpdateHandlerTestSuite))
}

func (s *SystemUpdateHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.conn = mock_events.NewMockChainConnection(ctrl)
	s.systemUpdateHandler = listener.NewSystemUpdateEventHandler(s.conn)
}

func (s *SystemUpdateHandlerTestSuite) Test_UpdateMetadataFails() {
	s.conn.EXPECT().UpdateMetatdata().Return(fmt.Errorf("error"))

	evtsRec := types.EventRecords{
		System_CodeUpdated: make([]types.EventSystemCodeUpdated, 1),
	}
	evts := events.Events{
		EventRecords:                evtsRec,
		SygmaBridge_Deposit:         []events.Deposit{},
		SygmaBasicFeeHandler_FeeSet: []events.FeeSet{},

		SygmaBridge_ProposalExecution:      []events.ProposalExecution{},
		SygmaBridge_FailedHandlerExecution: []events.FailedHandlerExecution{},
		SygmaBridge_Retry:                  []events.Retry{},
		SygmaBridge_BridgePaused:           []events.BridgePaused{},
		SygmaBridge_BridgeUnpaused:         []events.BridgeUnpaused{},
	}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(&evts, msgChan)

	s.NotNil(err)
	s.Equal(len(msgChan), 0)
}

func (s *SystemUpdateHandlerTestSuite) Test_NoMetadataUpdate() {
	evts := events.Events{}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(&evts, msgChan)

	s.Nil(err)
	s.Equal(len(msgChan), 0)
}

func (s *SystemUpdateHandlerTestSuite) Test_SuccesfullMetadataUpdate() {
	s.conn.EXPECT().UpdateMetatdata().Return(nil)

	evtsRec := types.EventRecords{
		System_CodeUpdated: make([]types.EventSystemCodeUpdated, 1),
	}
	evts := events.Events{EventRecords: evtsRec,
		SygmaBridge_Deposit:         []events.Deposit{},
		SygmaBasicFeeHandler_FeeSet: []events.FeeSet{},

		SygmaBridge_ProposalExecution:      []events.ProposalExecution{},
		SygmaBridge_FailedHandlerExecution: []events.FailedHandlerExecution{},
		SygmaBridge_Retry:                  []events.Retry{},
		SygmaBridge_BridgePaused:           []events.BridgePaused{},
		SygmaBridge_BridgeUnpaused:         []events.BridgeUnpaused{},
	}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(&evts, msgChan)

	s.Nil(err)
	s.Equal(len(msgChan), 0)
}

type DepositHandlerTestSuite struct {
	suite.Suite
	depositEventHandler *listener.FungibleTransferEventHandler
	mockDepositHandler  *mock_events.MockDepositHandler
	domainID            uint8
}

func TestRunDepositHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(DepositHandlerTestSuite))
}

func (s *DepositHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.domainID = 1
	s.mockDepositHandler = mock_events.NewMockDepositHandler(ctrl)
	s.depositEventHandler = listener.NewFungibleTransferEventHandler(s.domainID, s.mockDepositHandler)
}

func (s *DepositHandlerTestSuite) Test_HandleDepositFails_ExecutionContinue() {
	d1 := &events.Deposit{
		DepositNonce: 1,
		DestDomainID: 2,
		ResourceID:   types.Bytes32{1},
		TransferType: [1]byte{0},
		Handler:      [1]byte{0},
		CallData:     []byte{},
	}
	d2 := &events.Deposit{
		DepositNonce: 2,
		DestDomainID: 2,
		ResourceID:   types.Bytes32{1},
		TransferType: [1]byte{0},
		Handler:      [1]byte{0},
		CallData:     []byte{},
	}
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.CallData,
		d1.TransferType,
	).Return(&message.Message{}, fmt.Errorf("error"))

	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.CallData,
		d1.TransferType,
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	msgChan := make(chan []*message.Message, 2)
	evtsRec := types.EventRecords{
		System_CodeUpdated: make([]types.EventSystemCodeUpdated, 1),
	}
	evts := events.Events{EventRecords: evtsRec,
		SygmaBridge_Deposit: []events.Deposit{
			*d1, *d2,
		},
		SygmaBasicFeeHandler_FeeSet: []events.FeeSet{},

		SygmaBridge_ProposalExecution:      []events.ProposalExecution{},
		SygmaBridge_FailedHandlerExecution: []events.FailedHandlerExecution{},
		SygmaBridge_Retry:                  []events.Retry{},
		SygmaBridge_BridgePaused:           []events.BridgePaused{},
		SygmaBridge_BridgeUnpaused:         []events.BridgeUnpaused{},
	}
	err := s.depositEventHandler.HandleEvents(&evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 2}})
}

func (s *DepositHandlerTestSuite) Test_SuccessfulHandleDeposit() {
	d1 := &events.Deposit{
		DepositNonce: 1,
		DestDomainID: 2,
		ResourceID:   types.Bytes32{1},
		TransferType: [1]byte{0},
		Handler:      [1]byte{0},
		CallData:     []byte{},
	}
	d2 := &events.Deposit{
		DepositNonce: 2,
		DestDomainID: 2,
		ResourceID:   types.Bytes32{1},
		TransferType: [1]byte{0},
		Handler:      [1]byte{0},
		CallData:     []byte{},
	}
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.CallData,
		d1.TransferType,
	).Return(
		&message.Message{DepositNonce: 1},
		nil,
	)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.CallData,
		d2.TransferType,
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	msgChan := make(chan []*message.Message, 2)
	evtsRec := types.EventRecords{
		System_CodeUpdated: make([]types.EventSystemCodeUpdated, 1),
	}
	evts := events.Events{EventRecords: evtsRec,
		SygmaBridge_Deposit: []events.Deposit{
			*d1, *d2,
		},
		SygmaBasicFeeHandler_FeeSet: []events.FeeSet{},

		SygmaBridge_ProposalExecution:      []events.ProposalExecution{},
		SygmaBridge_FailedHandlerExecution: []events.FailedHandlerExecution{},
		SygmaBridge_Retry:                  []events.Retry{},
		SygmaBridge_BridgePaused:           []events.BridgePaused{},
		SygmaBridge_BridgeUnpaused:         []events.BridgeUnpaused{},
	}
	err := s.depositEventHandler.HandleEvents(&evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 1}, {DepositNonce: 2}})
}

func (s *DepositHandlerTestSuite) Test_HandleDepositPanics_ExecutionContinues() {
	d1 := &events.Deposit{
		DepositNonce: 1,
		DestDomainID: 2,
		ResourceID:   types.Bytes32{1},
		TransferType: [1]byte{0},
		Handler:      [1]byte{0},
		CallData:     []byte{},
	}
	d2 := &events.Deposit{
		DepositNonce: 2,
		DestDomainID: 2,
		ResourceID:   types.Bytes32{1},
		TransferType: [1]byte{0},
		Handler:      [1]byte{0},
		CallData:     []byte{},
	}
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.CallData,
		d1.TransferType,
	).Do(func(sourceID, destID, nonce, resourceID, calldata, depositType, handlerResponse interface{}) {
		panic("error")
	})
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d1.CallData,
		d1.TransferType,
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	msgChan := make(chan []*message.Message, 2)
	evtsRec := types.EventRecords{
		System_CodeUpdated: make([]types.EventSystemCodeUpdated, 1),
	}
	evts := events.Events{EventRecords: evtsRec,
		SygmaBridge_Deposit: []events.Deposit{
			*d1, *d2,
		},
		SygmaBasicFeeHandler_FeeSet: []events.FeeSet{},

		SygmaBridge_ProposalExecution:      []events.ProposalExecution{},
		SygmaBridge_FailedHandlerExecution: []events.FailedHandlerExecution{},
		SygmaBridge_Retry:                  []events.Retry{},
		SygmaBridge_BridgePaused:           []events.BridgePaused{},
		SygmaBridge_BridgeUnpaused:         []events.BridgeUnpaused{},
	}
	err := s.depositEventHandler.HandleEvents(&evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 2}})
}

type RetryHandlerTestSuite struct {
	suite.Suite
	mockDepositHandler *mock_events.MockDepositHandler
	mockConn           *mock_events.MockChainConnection
	domainID           uint8
}

func TestRunRetryHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RetryHandlerTestSuite))
}

func (s *RetryHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.domainID = 1
	s.mockDepositHandler = mock_events.NewMockDepositHandler(ctrl)
	s.mockConn = mock_events.NewMockChainConnection(ctrl)
}

func (s *RetryHandlerTestSuite) Test_CannotFetchLatestBlock() {
	s.mockConn.EXPECT().GetBlockLatest().Return(nil, fmt.Errorf("error"))

	retryHandler := listener.NewRetryEventHandler(s.mockConn, s.mockDepositHandler, s.domainID, big.NewInt(5))
	msgChan := make(chan []*message.Message, 2)
	err := retryHandler.HandleEvents([]*events.Events{}, msgChan)

	s.NotNil(err)
}

func (s *RetryHandlerTestSuite) Test_EventTooNew() {
	s.mockConn.EXPECT().GetBlockLatest().Return(&types.SignedBlock{Block: types.Block{
		Header: types.Header{
			Number: types.BlockNumber(uint32(100)),
		},
	}}, nil)

	retryHandler := listener.NewRetryEventHandler(s.mockConn, s.mockDepositHandler, s.domainID, big.NewInt(5))
	msgChan := make(chan []*message.Message)
	evts := []*events.Events{
		{
			SygmaBridge_Retry: []events.Retry{
				{
					DepositOnBlockHeight: types.NewU128(*big.NewInt(110)),
				},
			},
		},
	}
	err := retryHandler.HandleEvents(evts, msgChan)

	s.Nil(err)
	s.Equal(len(msgChan), 0)
}

func (s *RetryHandlerTestSuite) Test_FetchingBlockHashFails() {
	s.mockConn.EXPECT().GetBlockLatest().Return(&types.SignedBlock{Block: types.Block{
		Header: types.Header{
			Number: types.BlockNumber(uint32(100)),
		},
	}}, nil)
	s.mockConn.EXPECT().GetBlockHash(uint64(95)).Return(types.Hash{}, fmt.Errorf("error"))

	retryHandler := listener.NewRetryEventHandler(s.mockConn, s.mockDepositHandler, s.domainID, big.NewInt(5))
	msgChan := make(chan []*message.Message)
	evts := []*events.Events{
		{
			SygmaBridge_Retry: []events.Retry{
				{
					DepositOnBlockHeight: types.NewU128(*big.NewInt(95)),
				},
			},
		},
	}
	err := retryHandler.HandleEvents(evts, msgChan)

	s.NotNil(err)
	s.Equal(len(msgChan), 0)
}

func (s *RetryHandlerTestSuite) Test_FetchingBlockEventsFails() {
	s.mockConn.EXPECT().GetBlockLatest().Return(&types.SignedBlock{Block: types.Block{
		Header: types.Header{
			Number: types.BlockNumber(uint32(100)),
		},
	}}, nil)
	s.mockConn.EXPECT().GetBlockHash(uint64(95)).Return(types.Hash{}, nil)
	s.mockConn.EXPECT().GetBlockEvents(gomock.Any()).Return(nil, fmt.Errorf("error"))

	retryHandler := listener.NewRetryEventHandler(s.mockConn, s.mockDepositHandler, s.domainID, big.NewInt(5))
	msgChan := make(chan []*message.Message)
	evts := []*events.Events{
		{
			SygmaBridge_Retry: []events.Retry{
				{
					DepositOnBlockHeight: types.NewU128(*big.NewInt(95)),
				},
			},
		},
	}
	err := retryHandler.HandleEvents(evts, msgChan)

	s.NotNil(err)
	s.Equal(len(msgChan), 0)
}

func (s *RetryHandlerTestSuite) Test_NoEvents() {
	s.mockConn.EXPECT().GetBlockLatest().Return(&types.SignedBlock{Block: types.Block{
		Header: types.Header{
			Number: types.BlockNumber(uint32(100)),
		},
	}}, nil)
	s.mockConn.EXPECT().GetBlockHash(uint64(95)).Return(types.Hash{}, nil)
	s.mockConn.EXPECT().GetBlockEvents(gomock.Any()).Return(&events.Events{}, nil)

	retryHandler := listener.NewRetryEventHandler(s.mockConn, s.mockDepositHandler, s.domainID, big.NewInt(5))
	msgChan := make(chan []*message.Message)
	evts := []*events.Events{
		{
			SygmaBridge_Retry: []events.Retry{
				{
					DepositOnBlockHeight: types.NewU128(*big.NewInt(95)),
				},
			},
		},
	}
	err := retryHandler.HandleEvents(evts, msgChan)

	s.Nil(err)
	s.Equal(len(msgChan), 0)
}

func (s *RetryHandlerTestSuite) Test_ValidEvents() {
	s.mockConn.EXPECT().GetBlockLatest().Return(&types.SignedBlock{Block: types.Block{
		Header: types.Header{
			Number: types.BlockNumber(uint32(100)),
		},
	}}, nil)
	s.mockConn.EXPECT().GetBlockHash(uint64(95)).Return(types.Hash{}, nil)
	d1 := &events.Deposit{
		DepositNonce: 1,
		DestDomainID: 2,
		ResourceID:   types.Bytes32{1},
		TransferType: [1]byte{0},
		Handler:      [1]byte{0},
		CallData:     []byte{},
	}
	d2 := &events.Deposit{
		DepositNonce: 2,
		DestDomainID: 2,
		ResourceID:   types.Bytes32{1},
		TransferType: [1]byte{0},
		Handler:      [1]byte{0},
		CallData:     []byte{},
	}
	blockEvts := &events.Events{
		SygmaBridge_Deposit: []events.Deposit{
			*d1, *d2,
		},
	}
	s.mockConn.EXPECT().GetBlockEvents(gomock.Any()).Return(blockEvts, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.CallData,
		d1.TransferType,
	).Return(
		&message.Message{DepositNonce: 1},
		nil,
	)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.CallData,
		d2.TransferType,
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	retryHandler := listener.NewRetryEventHandler(s.mockConn, s.mockDepositHandler, s.domainID, big.NewInt(5))
	msgChan := make(chan []*message.Message, 2)
	evts := []*events.Events{
		{
			SygmaBridge_Retry: []events.Retry{
				{
					DepositOnBlockHeight: types.NewU128(*big.NewInt(95)),
				},
			},
		},
	}
	err := retryHandler.HandleEvents(evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(len(msgChan), 0)
	s.Equal(msgs, []*message.Message{{DepositNonce: 1}, {DepositNonce: 2}})
}

func (s *RetryHandlerTestSuite) Test_EventPanics() {
	s.mockConn.EXPECT().GetBlockLatest().Return(&types.SignedBlock{Block: types.Block{
		Header: types.Header{
			Number: types.BlockNumber(uint32(100)),
		},
	}}, nil)
	s.mockConn.EXPECT().GetBlockHash(uint64(95)).Return(types.Hash{}, nil)
	s.mockConn.EXPECT().GetBlockHash(uint64(95)).Return(types.Hash{}, nil)
	d1 := &events.Deposit{
		DepositNonce: 1,
		DestDomainID: 2,
		ResourceID:   types.Bytes32{1},
		TransferType: [1]byte{0},
		Handler:      [1]byte{0},
		CallData:     []byte{},
	}
	d2 := &events.Deposit{
		DepositNonce: 2,
		DestDomainID: 2,
		ResourceID:   types.Bytes32{1},
		TransferType: [1]byte{0},
		Handler:      [1]byte{0},
		CallData:     []byte{},
	}
	blockEvts1 := &events.Events{
		SygmaBridge_Deposit: []events.Deposit{
			*d1,
		},
	}
	blockEvts2 := &events.Events{
		SygmaBridge_Deposit: []events.Deposit{
			*d2,
		},
	}
	s.mockConn.EXPECT().GetBlockEvents(gomock.Any()).Return(blockEvts1, nil)
	s.mockConn.EXPECT().GetBlockEvents(gomock.Any()).Return(blockEvts2, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.CallData,
		d1.TransferType,
	).Do(func(sourceID, destID, nonce, resourceID, calldata, depositType interface{}) {
		panic("error")
	})
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.CallData,
		d2.TransferType,
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	retryHandler := listener.NewRetryEventHandler(s.mockConn, s.mockDepositHandler, s.domainID, big.NewInt(5))
	msgChan := make(chan []*message.Message, 1)
	evts := []*events.Events{
		{
			SygmaBridge_Retry: []events.Retry{
				{
					DepositOnBlockHeight: types.NewU128(*big.NewInt(95)),
				},
			},
		},
		{
			SygmaBridge_Retry: []events.Retry{
				{
					DepositOnBlockHeight: types.NewU128(*big.NewInt(95)),
				},
			},
		},
	}
	err := retryHandler.HandleEvents(evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(len(msgChan), 0)
	s.Equal(msgs, []*message.Message{{DepositNonce: 2}})
}
