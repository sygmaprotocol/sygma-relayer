// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener_test

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/listener"
	mock_events "github.com/ChainSafe/sygma-relayer/chains/substrate/listener/mock"
	"github.com/rs/zerolog"

	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/parser"
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

	evts := []*parser.Event{
		{
			Name: "ParachainSystem.ValidationFunctionApplied",
		},
	}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(evts, msgChan)

	s.NotNil(err)
	s.Equal(len(msgChan), 0)
}

func (s *SystemUpdateHandlerTestSuite) Test_NoMetadataUpdate() {
	evts := []*parser.Event{}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(evts, msgChan)

	s.Nil(err)
	s.Equal(len(msgChan), 0)
}

func (s *SystemUpdateHandlerTestSuite) Test_SuccesfullMetadataUpdate() {
	s.conn.EXPECT().UpdateMetatdata().Return(nil)
	evts := []*parser.Event{
		{
			Name: "ParachainSystem.ValidationFunctionApplied",
		},
	}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(evts, msgChan)

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
	s.depositEventHandler = listener.NewFungibleTransferEventHandler(zerolog.Context{}, s.domainID, s.mockDepositHandler)
}

func (s *DepositHandlerTestSuite) Test_HandleDepositFails_ExecutionContinue() {
	d1 := map[string]any{
		"dest_domain_id":            types.NewU8(2),
		"deposit_nonce":             types.NewU64(1),
		"resource_id":               types.Bytes32{1},
		"sygma_traits_TransferType": types.NewU8(0),
		"handler_response":          [1]byte{0},
		"deposit_data":              []byte{},
	}
	d2 := map[string]any{
		"deposit_nonce":             types.NewU64(2),
		"dest_domain_id":            types.NewU8(2),
		"resource_id":               types.Bytes32{1},
		"sygma_traits_TransferType": types.NewU8(0),
		"handler_response":          [1]byte{0},
		"deposit_data":              []byte{},
	}
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1["dest_domain_id"],
		d1["deposit_nonce"],
		d1["resource_id"],
		d1["deposit_data"],
		d1["sygma_traits_TransferType"],
	).Return(&message.Message{}, fmt.Errorf("error"))

	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2["dest_domain_id"],
		d2["deposit_nonce"],
		d2["resource_id"],
		d2["deposit_data"],
		d1["sygma_traits_TransferType"],
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	msgChan := make(chan []*message.Message, 2)
	evts := []*parser.Event{
		{
			Name:   "SygmaBridge.Deposit",
			Fields: d1,
		},
		{
			Name:   "SygmaBridge.Deposit",
			Fields: d2,
		},
	}
	err := s.depositEventHandler.HandleEvents(evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 2}})
}

func (s *DepositHandlerTestSuite) Test_SuccessfulHandleDeposit() {
	d1 := map[string]any{
		"dest_domain_id":            types.NewU8(2),
		"deposit_nonce":             types.NewU64(1),
		"resource_id":               types.Bytes32{1},
		"sygma_traits_TransferType": types.NewU8(0),
		"handler_response":          [1]byte{0},
		"deposit_data":              []byte{},
	}
	d2 := map[string]any{
		"deposit_nonce":             types.NewU64(2),
		"dest_domain_id":            types.NewU8(2),
		"resource_id":               types.Bytes32{1},
		"sygma_traits_TransferType": types.NewU8(0),
		"handler_response":          [1]byte{0},
		"deposit_data":              []byte{},
	}
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1["dest_domain_id"],
		d1["deposit_nonce"],
		d1["resource_id"],
		d1["deposit_data"],
		d1["sygma_traits_TransferType"],
	).Return(
		&message.Message{DepositNonce: 1},
		nil,
	)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2["dest_domain_id"],
		d2["deposit_nonce"],
		d2["resource_id"],
		d2["deposit_data"],
		d2["sygma_traits_TransferType"],
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	msgChan := make(chan []*message.Message, 2)

	evts := []*parser.Event{
		{
			Name:   "SygmaBridge.Deposit",
			Fields: d1,
		},
		{
			Name:   "SygmaBridge.Deposit",
			Fields: d2,
		},
	}
	err := s.depositEventHandler.HandleEvents(evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 1}, {DepositNonce: 2}})
}

func (s *DepositHandlerTestSuite) Test_HandleDepositPanics_ExecutionContinues() {
	d1 := map[string]any{
		"dest_domain_id":            types.NewU8(2),
		"deposit_nonce":             types.NewU64(1),
		"resource_id":               types.Bytes32{1},
		"sygma_traits_TransferType": types.NewU8(0),
		"handler_response":          [1]byte{0},
		"deposit_data":              []byte{},
	}
	d2 := map[string]any{
		"deposit_nonce":             types.NewU64(2),
		"dest_domain_id":            types.NewU8(2),
		"resource_id":               types.Bytes32{1},
		"sygma_traits_TransferType": types.NewU8(0),
		"handler_response":          [1]byte{0},
		"deposit_data":              []byte{},
	}
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1["dest_domain_id"],
		d1["deposit_nonce"],
		d1["resource_id"],
		d1["deposit_data"],
		d1["sygma_traits_TransferType"],
	).Do(func(sourceID, destID, nonce, resourceID, calldata, depositType interface{}) {
		panic("error")
	})
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2["dest_domain_id"],
		d2["deposit_nonce"],
		d2["resource_id"],
		d2["deposit_data"],
		d2["sygma_traits_TransferType"],
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	msgChan := make(chan []*message.Message, 2)
	evts := []*parser.Event{
		{
			Name:   "SygmaBridge.Deposit",
			Fields: d1,
		},
		{
			Name:   "SygmaBridge.Deposit",
			Fields: d2,
		},
	}
	err := s.depositEventHandler.HandleEvents(evts, msgChan)
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
	s.mockConn.EXPECT().GetFinalizedHead().Return(types.Hash{}, fmt.Errorf("error"))

	retryHandler := listener.NewRetryEventHandler(zerolog.Context{}, s.mockConn, s.mockDepositHandler, s.domainID)
	msgChan := make(chan []*message.Message, 2)
	err := retryHandler.HandleEvents([]*parser.Event{}, msgChan)

	s.NotNil(err)
}

func (s *RetryHandlerTestSuite) Test_EventTooNew() {
	s.mockConn.EXPECT().GetFinalizedHead().Return(types.Hash{}, nil)
	s.mockConn.EXPECT().GetBlock(gomock.Any()).Return(&types.SignedBlock{Block: types.Block{
		Header: types.Header{
			Number: types.BlockNumber(uint32(100)),
		},
	}}, nil)

	retryHandler := listener.NewRetryEventHandler(zerolog.Context{}, s.mockConn, s.mockDepositHandler, s.domainID)
	msgChan := make(chan []*message.Message)
	rtry := map[string]any{
		"deposit_on_block_height": types.NewU128(*big.NewInt(101)),
	}
	evts := []*parser.Event{
		{
			Name:   "SygmaBridge.Retry",
			Fields: rtry,
		},
	}

	err := retryHandler.HandleEvents(evts, msgChan)

	s.Nil(err)
	s.Equal(len(msgChan), 0)
}

func (s *RetryHandlerTestSuite) Test_FetchingBlockHashFails() {
	s.mockConn.EXPECT().GetFinalizedHead().Return(types.Hash{}, nil)
	s.mockConn.EXPECT().GetBlock(gomock.Any()).Return(&types.SignedBlock{Block: types.Block{
		Header: types.Header{
			Number: types.BlockNumber(uint32(100)),
		},
	}}, nil)
	s.mockConn.EXPECT().GetBlockHash(uint64(95)).Return(types.Hash{}, fmt.Errorf("error"))

	retryHandler := listener.NewRetryEventHandler(zerolog.Context{}, s.mockConn, s.mockDepositHandler, s.domainID)
	msgChan := make(chan []*message.Message)
	rtry := map[string]any{
		"deposit_on_block_height": types.NewU128(*big.NewInt(95)),
	}
	evts := []*parser.Event{
		{
			Name:   "SygmaBridge.Retry",
			Fields: rtry,
		},
	}
	err := retryHandler.HandleEvents(evts, msgChan)

	s.NotNil(err)
	s.Equal(len(msgChan), 0)
}

func (s *RetryHandlerTestSuite) Test_FetchingBlockEventsFails() {
	s.mockConn.EXPECT().GetFinalizedHead().Return(types.Hash{}, nil)
	s.mockConn.EXPECT().GetBlock(gomock.Any()).Return(&types.SignedBlock{Block: types.Block{
		Header: types.Header{
			Number: types.BlockNumber(uint32(100)),
		},
	}}, nil)
	s.mockConn.EXPECT().GetBlockHash(uint64(95)).Return(types.Hash{}, nil)
	s.mockConn.EXPECT().GetBlockEvents(gomock.Any()).Return(nil, fmt.Errorf("error"))

	retryHandler := listener.NewRetryEventHandler(zerolog.Context{}, s.mockConn, s.mockDepositHandler, s.domainID)
	msgChan := make(chan []*message.Message)
	rtry := map[string]any{
		"deposit_on_block_height": types.NewU128(*big.NewInt(95)),
	}
	evts := []*parser.Event{
		{
			Name:   "SygmaBridge.Retry",
			Fields: rtry,
		},
	}
	err := retryHandler.HandleEvents(evts, msgChan)

	s.NotNil(err)
	s.Equal(len(msgChan), 0)
}

func (s *RetryHandlerTestSuite) Test_NoEvents() {
	s.mockConn.EXPECT().GetFinalizedHead().Return(types.Hash{}, nil)
	s.mockConn.EXPECT().GetBlock(gomock.Any()).Return(&types.SignedBlock{Block: types.Block{
		Header: types.Header{
			Number: types.BlockNumber(uint32(100)),
		},
	}}, nil)
	s.mockConn.EXPECT().GetBlockHash(uint64(95)).Return(types.Hash{}, nil)
	s.mockConn.EXPECT().GetBlockEvents(gomock.Any()).Return([]*parser.Event{}, nil)

	retryHandler := listener.NewRetryEventHandler(zerolog.Context{}, s.mockConn, s.mockDepositHandler, s.domainID)
	msgChan := make(chan []*message.Message)
	rtry := map[string]any{
		"deposit_on_block_height": types.NewU128(*big.NewInt(95)),
	}
	evts := []*parser.Event{
		{
			Name:   "SygmaBridge.Retry",
			Fields: rtry,
		},
	}
	err := retryHandler.HandleEvents(evts, msgChan)

	s.Nil(err)
	s.Equal(len(msgChan), 0)
}

func (s *RetryHandlerTestSuite) Test_ValidEvents() {
	s.mockConn.EXPECT().GetFinalizedHead().Return(types.Hash{}, nil)
	s.mockConn.EXPECT().GetBlock(gomock.Any()).Return(&types.SignedBlock{Block: types.Block{
		Header: types.Header{
			Number: types.BlockNumber(uint32(100)),
		},
	}}, nil)
	s.mockConn.EXPECT().GetBlockHash(uint64(95)).Return(types.Hash{}, nil)
	d1 := map[string]any{
		"dest_domain_id":            types.NewU8(2),
		"deposit_nonce":             types.NewU64(1),
		"resource_id":               types.Bytes32{1},
		"sygma_traits_TransferType": types.NewU8(0),
		"handler_response":          [1]byte{0},
		"deposit_data":              []byte{},
	}
	d2 := map[string]any{
		"deposit_nonce":             types.NewU64(2),
		"dest_domain_id":            types.NewU8(2),
		"resource_id":               types.Bytes32{1},
		"sygma_traits_TransferType": types.NewU8(0),
		"handler_response":          [1]byte{0},
		"deposit_data":              []byte{},
	}
	blockEvts := []*parser.Event{
		{
			Name:   "SygmaBridge.Deposit",
			Fields: d1,
		},
		{
			Name:   "SygmaBridge.Deposit",
			Fields: d2,
		},
	}
	s.mockConn.EXPECT().GetBlockEvents(gomock.Any()).Return(blockEvts, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1["dest_domain_id"],
		d1["deposit_nonce"],
		d1["resource_id"],
		d1["deposit_data"],
		d1["sygma_traits_TransferType"],
	).Return(
		&message.Message{DepositNonce: 1},
		nil,
	)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2["dest_domain_id"],
		d2["deposit_nonce"],
		d2["resource_id"],
		d2["deposit_data"],
		d2["sygma_traits_TransferType"],
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	retryHandler := listener.NewRetryEventHandler(zerolog.Context{}, s.mockConn, s.mockDepositHandler, s.domainID)
	msgChan := make(chan []*message.Message, 2)
	rtry := map[string]any{
		"deposit_on_block_height": types.NewU128(*big.NewInt(95)),
	}
	evts := []*parser.Event{
		{
			Name:   "SygmaBridge.Retry",
			Fields: rtry,
		},
	}
	err := retryHandler.HandleEvents(evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(len(msgChan), 0)
	s.Equal(msgs, []*message.Message{{DepositNonce: 1}, {DepositNonce: 2}})
}

func (s *RetryHandlerTestSuite) Test_EventPanics() {
	s.mockConn.EXPECT().GetFinalizedHead().Return(types.Hash{}, nil)
	s.mockConn.EXPECT().GetBlock(gomock.Any()).Return(&types.SignedBlock{Block: types.Block{
		Header: types.Header{
			Number: types.BlockNumber(uint32(100)),
		},
	}}, nil)
	s.mockConn.EXPECT().GetBlockHash(uint64(95)).Return(types.Hash{}, nil)
	s.mockConn.EXPECT().GetBlockHash(uint64(95)).Return(types.Hash{}, nil)
	d1 := map[string]any{
		"dest_domain_id":            types.NewU8(2),
		"deposit_nonce":             types.NewU64(1),
		"resource_id":               types.Bytes32{1},
		"sygma_traits_TransferType": types.NewU8(0),
		"handler_response":          [1]byte{0},
		"deposit_data":              []byte{},
	}
	d2 := map[string]any{
		"deposit_nonce":             types.NewU64(2),
		"dest_domain_id":            types.NewU8(2),
		"resource_id":               types.Bytes32{1},
		"sygma_traits_TransferType": types.NewU8(0),
		"handler_response":          [1]byte{0},
		"deposit_data":              []byte{},
	}

	blockEvts1 := []*parser.Event{
		{
			Name:   "SygmaBridge.Deposit",
			Fields: d1,
		},
	}
	blockEvts2 := []*parser.Event{
		{
			Name:   "SygmaBridge.Deposit",
			Fields: d2,
		},
	}
	s.mockConn.EXPECT().GetBlockEvents(gomock.Any()).Return(blockEvts1, nil)
	s.mockConn.EXPECT().GetBlockEvents(gomock.Any()).Return(blockEvts2, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1["dest_domain_id"],
		d1["deposit_nonce"],
		d1["resource_id"],
		d1["deposit_data"],
		d1["sygma_traits_TransferType"],
	).Do(func(sourceID, destID, nonce, resourceID, calldata, depositType interface{}) {
		panic("error")
	})
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2["dest_domain_id"],
		d2["deposit_nonce"],
		d2["resource_id"],
		d2["deposit_data"],
		d2["sygma_traits_TransferType"],
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	retryHandler := listener.NewRetryEventHandler(zerolog.Context{}, s.mockConn, s.mockDepositHandler, s.domainID)
	msgChan := make(chan []*message.Message, 1)
	rtry := map[string]any{
		"deposit_on_block_height": types.NewU128(*big.NewInt(95)),
	}
	evts := []*parser.Event{
		{
			Name:   "SygmaBridge.Retry",
			Fields: rtry,
		},
		{
			Name:   "SygmaBridge.Retry",
			Fields: rtry,
		},
	}
	err := retryHandler.HandleEvents(evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(len(msgChan), 0)
	s.Equal(msgs, []*message.Message{{DepositNonce: 2}})
}
