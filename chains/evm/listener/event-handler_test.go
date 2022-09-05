package listener_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	coreEvents "github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/types"

	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/events"
	"github.com/ChainSafe/sygma-relayer/chains/evm/listener"
	mock_listener "github.com/ChainSafe/sygma-relayer/chains/evm/listener/mock"
)

type RetryEventHandlerTestSuite struct {
	suite.Suite
	retryEventHandler  *listener.RetryEventHandler
	mockDepositHandler *mock_listener.MockDepositHandler
	mockEventListener  *mock_listener.MockEventListener
	domainID           uint8
}

func TestRunRetryEventHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RetryEventHandlerTestSuite))
}

func (s *RetryEventHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.domainID = 1
	s.mockEventListener = mock_listener.NewMockEventListener(ctrl)
	s.mockDepositHandler = mock_listener.NewMockDepositHandler(ctrl)
	s.retryEventHandler = listener.NewRetryEventHandler(s.mockEventListener, s.mockDepositHandler, common.Address{}, s.domainID, big.NewInt(5))
}

func (s *RetryEventHandlerTestSuite) Test_FetchDepositFails() {
	s.mockEventListener.EXPECT().FetchRetryEvents(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]events.RetryEvent{}, fmt.Errorf("error"))

	msgChan := make(chan []*message.Message, 1)
	err := s.retryEventHandler.HandleEvent(big.NewInt(0), big.NewInt(5), msgChan)

	s.NotNil(err)
	s.Equal(len(msgChan), 0)
}

func (s *RetryEventHandlerTestSuite) Test_FetchDepositFails_ExecutionContinues() {
	d := coreEvents.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	s.mockEventListener.EXPECT().FetchRetryEvents(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return([]events.RetryEvent{{TxHash: "event1"}, {TxHash: "event2"}}, nil)
	s.mockEventListener.EXPECT().FetchDepositEvent(events.RetryEvent{TxHash: "event1"}, gomock.Any(), big.NewInt(5)).Return([]coreEvents.Deposit{}, fmt.Errorf("error"))
	s.mockEventListener.EXPECT().FetchDepositEvent(events.RetryEvent{TxHash: "event2"}, gomock.Any(), big.NewInt(5)).Return([]coreEvents.Deposit{d}, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d.DestinationDomainID,
		d.DepositNonce,
		d.ResourceID,
		d.Data,
		d.HandlerResponse,
	).Return(&message.Message{DepositNonce: 2}, nil)

	msgChan := make(chan []*message.Message, 2)
	err := s.retryEventHandler.HandleEvent(big.NewInt(0), big.NewInt(5), msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 2}})
}

func (s *RetryEventHandlerTestSuite) Test_HandleDepositFails_ExecutionContinues() {
	d1 := coreEvents.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := coreEvents.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	s.mockEventListener.EXPECT().FetchRetryEvents(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return([]events.RetryEvent{{TxHash: "event1"}, {TxHash: "event2"}}, nil)
	s.mockEventListener.EXPECT().FetchDepositEvent(events.RetryEvent{TxHash: "event1"}, gomock.Any(), big.NewInt(5)).Return([]coreEvents.Deposit{d1}, nil)
	s.mockEventListener.EXPECT().FetchDepositEvent(events.RetryEvent{TxHash: "event2"}, gomock.Any(), big.NewInt(5)).Return([]coreEvents.Deposit{d2}, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.HandlerResponse,
	).Return(&message.Message{DepositNonce: 1}, fmt.Errorf("error"))
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.HandlerResponse,
	).Return(&message.Message{DepositNonce: 2}, nil)

	msgChan := make(chan []*message.Message, 2)
	err := s.retryEventHandler.HandleEvent(big.NewInt(0), big.NewInt(5), msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 2}})
}

func (s *RetryEventHandlerTestSuite) Test_HandlingRetryPanics_ExecutionContinue() {
	d1 := coreEvents.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := coreEvents.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	s.mockEventListener.EXPECT().FetchRetryEvents(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return([]events.RetryEvent{{TxHash: "event1"}, {TxHash: "event2"}}, nil)
	s.mockEventListener.EXPECT().FetchDepositEvent(events.RetryEvent{TxHash: "event1"}, gomock.Any(), big.NewInt(5)).Return([]coreEvents.Deposit{d1}, nil)
	s.mockEventListener.EXPECT().FetchDepositEvent(events.RetryEvent{TxHash: "event2"}, gomock.Any(), big.NewInt(5)).Return([]coreEvents.Deposit{d2}, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.HandlerResponse,
	).Do(func(sourceID, destID, nonce, resourceID, calldata, handlerResponse interface{}) {
		panic("error")
	})
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.HandlerResponse,
	).Return(&message.Message{DepositNonce: 2}, nil)

	msgChan := make(chan []*message.Message, 2)
	err := s.retryEventHandler.HandleEvent(big.NewInt(0), big.NewInt(5), msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 2}})
}

func (s *RetryEventHandlerTestSuite) Test_MultipleDeposits() {
	d1 := coreEvents.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := coreEvents.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	s.mockEventListener.EXPECT().FetchRetryEvents(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return([]events.RetryEvent{{TxHash: "event1"}}, nil)
	s.mockEventListener.EXPECT().FetchDepositEvent(events.RetryEvent{TxHash: "event1"}, gomock.Any(), big.NewInt(5)).Return([]coreEvents.Deposit{d1, d2}, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.HandlerResponse,
	).Return(&message.Message{DepositNonce: 1}, nil)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.HandlerResponse,
	).Return(&message.Message{DepositNonce: 2}, nil)

	msgChan := make(chan []*message.Message, 2)
	err := s.retryEventHandler.HandleEvent(big.NewInt(0), big.NewInt(5), msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 1}, {DepositNonce: 2}})
}
