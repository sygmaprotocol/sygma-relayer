// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package eventHandlers_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/rs/zerolog/log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/events"
	"github.com/ChainSafe/sygma-relayer/chains/evm/listener/eventHandlers"
	mock_listener "github.com/ChainSafe/sygma-relayer/chains/evm/listener/eventHandlers/mock"
	"github.com/ChainSafe/sygma-relayer/relayer/transfer"
	"github.com/ChainSafe/sygma-relayer/store"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type RetryEventHandlerTestSuite struct {
	suite.Suite
	retryEventHandler  *eventHandlers.RetryEventHandler
	mockDepositHandler *mock_listener.MockDepositHandler
	mockPropStorer     *mock_listener.MockPropStorer
	mockEventListener  *mock_listener.MockEventListener
	domainID           uint8
	msgChan            chan []*message.Message
}

func TestRunRetryEventHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RetryEventHandlerTestSuite))
}

func (s *RetryEventHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.domainID = 1
	s.mockEventListener = mock_listener.NewMockEventListener(ctrl)
	s.mockDepositHandler = mock_listener.NewMockDepositHandler(ctrl)
	s.mockPropStorer = mock_listener.NewMockPropStorer(ctrl)
	s.msgChan = make(chan []*message.Message, 1)
	s.retryEventHandler = eventHandlers.NewRetryEventHandler(
		log.With(),
		s.mockEventListener,
		s.mockDepositHandler,
		s.mockPropStorer,
		common.Address{},
		s.domainID,
		big.NewInt(5),
		s.msgChan)
}

func (s *RetryEventHandlerTestSuite) Test_FetchDepositFails() {
	s.mockEventListener.EXPECT().FetchRetryEvents(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]events.RetryEvent{}, fmt.Errorf("error"))

	err := s.retryEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))

	s.NotNil(err)
	s.Equal(len(s.msgChan), 0)
}

func (s *RetryEventHandlerTestSuite) Test_FetchDepositFails_ExecutionContinues() {
	d := events.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	msgID := fmt.Sprintf("retry-%d-%d-%d-%d", 1, 2, 0, 5)
	s.mockEventListener.EXPECT().FetchRetryEvents(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return([]events.RetryEvent{{TxHash: "event1"}, {TxHash: "event2"}}, nil)
	s.mockEventListener.EXPECT().FetchRetryDepositEvents(events.RetryEvent{TxHash: "event1"}, gomock.Any(), big.NewInt(5)).Return([]events.Deposit{}, fmt.Errorf("error"))
	s.mockEventListener.EXPECT().FetchRetryDepositEvents(events.RetryEvent{TxHash: "event2"}, gomock.Any(), big.NewInt(5)).Return([]events.Deposit{d}, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d.DestinationDomainID,
		d.DepositNonce,
		d.ResourceID,
		d.Data,
		d.HandlerResponse,
		msgID,
	).Return(&message.Message{
		Data: transfer.TransferMessageData{
			DepositNonce: 2,
		},
	}, nil)
	s.mockPropStorer.EXPECT().PropStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(store.MissingProp, nil)

	err := s.retryEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))
	msgs := <-s.msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{Data: transfer.TransferMessageData{
		DepositNonce: 2,
	}}})
}

func (s *RetryEventHandlerTestSuite) Test_HandleDepositFails_ExecutionContinues() {
	d1 := events.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := events.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	msgID := fmt.Sprintf("retry-%d-%d-%d-%d", 1, 2, 0, 5)
	s.mockEventListener.EXPECT().FetchRetryEvents(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return([]events.RetryEvent{{TxHash: "event1"}, {TxHash: "event2"}}, nil)
	s.mockEventListener.EXPECT().FetchRetryDepositEvents(events.RetryEvent{TxHash: "event1"}, gomock.Any(), big.NewInt(5)).Return([]events.Deposit{d1}, nil)
	s.mockEventListener.EXPECT().FetchRetryDepositEvents(events.RetryEvent{TxHash: "event2"}, gomock.Any(), big.NewInt(5)).Return([]events.Deposit{d2}, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.HandlerResponse,
		msgID,
	).Return(&message.Message{Data: transfer.TransferMessageData{
		DepositNonce: 1,
	}}, fmt.Errorf("error"))
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.HandlerResponse,
		msgID,
	).Return(&message.Message{Data: transfer.TransferMessageData{
		DepositNonce: 2,
	}}, nil)
	s.mockPropStorer.EXPECT().PropStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(store.MissingProp, nil)

	err := s.retryEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))
	msgs := <-s.msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{Data: transfer.TransferMessageData{DepositNonce: 2}}})
}

func (s *RetryEventHandlerTestSuite) Test_HandlingRetryPanics_ExecutionContinue() {
	d1 := events.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := events.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	s.mockEventListener.EXPECT().FetchRetryEvents(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return([]events.RetryEvent{{TxHash: "event1"}, {TxHash: "event2"}}, nil)
	s.mockEventListener.EXPECT().FetchRetryDepositEvents(events.RetryEvent{TxHash: "event1"}, gomock.Any(), big.NewInt(5)).Return([]events.Deposit{d1}, nil)
	s.mockEventListener.EXPECT().FetchRetryDepositEvents(events.RetryEvent{TxHash: "event2"}, gomock.Any(), big.NewInt(5)).Return([]events.Deposit{d2}, nil)
	msgID := fmt.Sprintf("retry-%d-%d-%d-%d", 1, 2, 0, 5)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.HandlerResponse,
		msgID,
	).Do(func(sourceID, destID, nonce, resourceID, calldata, handlerResponse, msgID interface{}) {
		panic("error")
	})
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.HandlerResponse,
		msgID,
	).Return(&message.Message{Data: transfer.TransferMessageData{
		DepositNonce: 2,
	}}, nil)
	s.mockPropStorer.EXPECT().PropStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(store.MissingProp, nil)

	err := s.retryEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))
	msgs := <-s.msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{Data: transfer.TransferMessageData{
		DepositNonce: 2,
	}}})
}

func (s *RetryEventHandlerTestSuite) Test_MultipleDeposits() {
	d1 := events.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := events.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	s.mockEventListener.EXPECT().FetchRetryEvents(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return([]events.RetryEvent{{TxHash: "event1"}}, nil)
	s.mockEventListener.EXPECT().FetchRetryDepositEvents(events.RetryEvent{TxHash: "event1"}, gomock.Any(), big.NewInt(5)).Return([]events.Deposit{d1, d2}, nil)
	msgID := fmt.Sprintf("retry-%d-%d-%d-%d", 1, 2, 0, 5)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.HandlerResponse,
		msgID,
	).Return(&message.Message{Data: transfer.TransferMessageData{
		DepositNonce: 1,
	}}, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.HandlerResponse,
		msgID,
	).Return(&message.Message{Data: transfer.TransferMessageData{
		DepositNonce: 2,
	}}, nil)
	s.mockPropStorer.EXPECT().PropStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(store.MissingProp, nil).Times(2)

	err := s.retryEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))
	msgs := <-s.msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{Data: transfer.TransferMessageData{
		DepositNonce: 1,
	}}, {Data: transfer.TransferMessageData{
		DepositNonce: 2,
	}}})
}

func (s *RetryEventHandlerTestSuite) Test_MultipleDeposits_ExecutedIgnored() {
	d1 := events.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := events.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	s.mockEventListener.EXPECT().FetchRetryEvents(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return([]events.RetryEvent{{TxHash: "event1"}}, nil)
	s.mockEventListener.EXPECT().FetchRetryDepositEvents(events.RetryEvent{TxHash: "event1"}, gomock.Any(), big.NewInt(5)).Return([]events.Deposit{d1, d2}, nil)
	msgID := fmt.Sprintf("retry-%d-%d-%d-%d", 1, 2, 0, 5)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.HandlerResponse,
		msgID,
	).Return(&message.Message{Data: transfer.TransferMessageData{
		DepositNonce: 1,
	}}, nil)
	s.mockPropStorer.EXPECT().PropStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(store.ExecutedProp, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.HandlerResponse,
		msgID,
	).Return(&message.Message{Data: transfer.TransferMessageData{
		DepositNonce: 2,
	}}, nil)
	s.mockPropStorer.EXPECT().PropStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(store.PendingProp, nil)
	s.mockPropStorer.EXPECT().StorePropStatus(gomock.Any(), gomock.Any(), gomock.Any(), store.FailedProp).Return(nil)

	err := s.retryEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))
	msgs := <-s.msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{
		{
			Data: transfer.TransferMessageData{
				DepositNonce: 2,
			},
		},
	})
}

type DepositHandlerTestSuite struct {
	suite.Suite
	depositEventHandler *eventHandlers.DepositEventHandler
	mockDepositHandler  *mock_listener.MockDepositHandler
	mockEventListener   *mock_listener.MockEventListener
	domainID            uint8
	msgChan             chan []*message.Message
}

func TestRunDepositHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(DepositHandlerTestSuite))
}

func (s *DepositHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.domainID = 1
	s.mockEventListener = mock_listener.NewMockEventListener(ctrl)
	s.mockDepositHandler = mock_listener.NewMockDepositHandler(ctrl)
	s.msgChan = make(chan []*message.Message, 2)
	s.depositEventHandler = eventHandlers.NewDepositEventHandler(s.mockEventListener, s.mockDepositHandler, common.Address{}, s.domainID, s.msgChan)
}

func (s *DepositHandlerTestSuite) Test_FetchDepositFails() {
	s.mockEventListener.EXPECT().FetchDeposits(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*events.Deposit{}, fmt.Errorf("error"))

	err := s.depositEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))

	s.NotNil(err)
	s.Equal(len(s.msgChan), 0)
}

func (s *DepositHandlerTestSuite) Test_HandleDepositFails_ExecutionContinue() {
	d1 := &events.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := &events.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	deposits := []*events.Deposit{d1, d2}
	s.mockEventListener.EXPECT().FetchDeposits(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(deposits, nil)
	msgID := fmt.Sprintf("%d-%d-%d-%d", 1, 2, 0, 5)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.HandlerResponse,
		msgID,
	).Return(&message.Message{}, fmt.Errorf("error"))
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.HandlerResponse,
		msgID,
	).Return(
		&message.Message{
			Data: transfer.TransferMessageData{
				DepositNonce: 2,
			},
		},
		nil,
	)

	err := s.depositEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))
	msgs := <-s.msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{Data: transfer.TransferMessageData{DepositNonce: 2}}})
}

func (s *DepositHandlerTestSuite) Test_HandleDepositPanis_ExecutionContinues() {
	d1 := &events.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := &events.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	deposits := []*events.Deposit{d1, d2}
	s.mockEventListener.EXPECT().FetchDeposits(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(deposits, nil)
	msgID := fmt.Sprintf("%d-%d-%d-%d", 1, 2, 0, 5)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.HandlerResponse,
		msgID,
	).Do(func(sourceID, destID, nonce, resourceID, calldata, handlerResponse, msgID interface{}) {
		panic("error")
	})
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.HandlerResponse,
		msgID,
	).Return(
		&message.Message{Data: transfer.TransferMessageData{DepositNonce: 2}},
		nil,
	)

	err := s.depositEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))
	msgs := <-s.msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{Data: transfer.TransferMessageData{DepositNonce: 2}}})
}

func (s *DepositHandlerTestSuite) Test_SuccessfulHandleDeposit() {
	d1 := &events.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := &events.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          [32]byte{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	deposits := []*events.Deposit{d1, d2}
	s.mockEventListener.EXPECT().FetchDeposits(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(deposits, nil)
	msgID := fmt.Sprintf("%d-%d-%d-%d", 1, 2, 0, 5)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.HandlerResponse,
		msgID,
	).Return(
		&message.Message{Data: transfer.TransferMessageData{DepositNonce: 1}},
		nil,
	)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.HandlerResponse,
		msgID,
	).Return(
		&message.Message{Data: transfer.TransferMessageData{DepositNonce: 2}},
		nil,
	)

	err := s.depositEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))
	msgs := <-s.msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{Data: transfer.TransferMessageData{DepositNonce: 1}}, {Data: transfer.TransferMessageData{DepositNonce: 2}}})
}

type TransferLiqudityHandlerTestSuite struct {
	suite.Suite
	transferLiqudityEventHandler *eventHandlers.TransferLiqudityEventHandler
	mockEventListener            *mock_listener.MockEventListener
	domainID                     uint8
	msgChan                      chan []*message.Message
}

func TestRunTrasferLiqudityHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(TransferLiqudityHandlerTestSuite))
}

func (s *TransferLiqudityHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.domainID = 1
	s.mockEventListener = mock_listener.NewMockEventListener(ctrl)
	s.msgChan = make(chan []*message.Message, 2)
	s.transferLiqudityEventHandler = eventHandlers.NewTransferLiquidityEventHandler(log.Logger.With(), s.mockEventListener, common.Address{}, s.domainID, s.msgChan)
}

func (s *TransferLiqudityHandlerTestSuite) Test_FetchDepositFails() {
	s.mockEventListener.EXPECT().FetchTransferLiqudityEvents(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*events.TransferLiquidity{}, fmt.Errorf("error"))

	err := s.transferLiqudityEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))

	s.NotNil(err)
	s.Equal(len(s.msgChan), 0)
}

func (s *TransferLiqudityHandlerTestSuite) Test_NoEvents() {
	s.mockEventListener.EXPECT().FetchTransferLiqudityEvents(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*events.TransferLiquidity{}, nil)

	err := s.transferLiqudityEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))

	s.Nil(err)
	s.Equal(len(s.msgChan), 0)
}

func (s *TransferLiqudityHandlerTestSuite) Test_ValidEvents() {
	e1 := &events.TransferLiquidity{
		DomainID:           2,
		ResourceID:         [32]byte{1},
		Amount:             big.NewInt(100),
		TransactionHash:    "hash1",
		DestinationAddress: []byte("recipient1"),
	}
	e2 := &events.TransferLiquidity{
		DomainID:           3,
		ResourceID:         [32]byte{2},
		Amount:             big.NewInt(200),
		TransactionHash:    "hash2",
		DestinationAddress: []byte("recipient2"),
	}
	s.mockEventListener.EXPECT().FetchTransferLiqudityEvents(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*events.TransferLiquidity{e1, e2}, nil)

	err := s.transferLiqudityEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))

	s.Nil(err)
	msg1 := <-s.msgChan
	msg2 := <-s.msgChan

	s.Equal(len(msg1), 1)
	s.Equal(len(msg2), 1)
	s.Equal(msg1[0].Data.(transfer.TransferMessageData), transfer.TransferMessageData{
		DepositNonce: chains.CalculateNonce(big.NewInt(5), e1.TransactionHash),
		ResourceId:   e1.ResourceID,
		Metadata:     nil,
		Payload: []interface{}{
			common.LeftPadBytes(e1.Amount.Bytes(), 32),
			e1.DestinationAddress,
		},
		Type: transfer.FungibleTransfer,
	})
	s.Equal(msg2[0].Data.(transfer.TransferMessageData), transfer.TransferMessageData{
		DepositNonce: chains.CalculateNonce(big.NewInt(5), e2.TransactionHash),
		ResourceId:   e2.ResourceID,
		Metadata:     nil,
		Payload: []interface{}{
			common.LeftPadBytes(e2.Amount.Bytes(), 32),
			e2.DestinationAddress,
		},
		Type: transfer.FungibleTransfer,
	})

}
