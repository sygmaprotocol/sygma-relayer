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

	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/events"
	"github.com/ChainSafe/sygma-relayer/chains/evm/listener/eventHandlers"
	mock_listener "github.com/ChainSafe/sygma-relayer/chains/evm/listener/eventHandlers/mock"
	"github.com/ChainSafe/sygma-relayer/relayer/retry"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type RetryEventHandlerTestSuite struct {
	suite.Suite
	retryEventHandler *eventHandlers.RetryEventHandler
	mockEventListener *mock_listener.MockEventListener
	domainID          uint8
	msgChan           chan []*message.Message
}

func TestRunRetryEventHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RetryEventHandlerTestSuite))
}

func (s *RetryEventHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.domainID = 1
	s.mockEventListener = mock_listener.NewMockEventListener(ctrl)
	s.msgChan = make(chan []*message.Message, 1)
	s.retryEventHandler = eventHandlers.NewRetryEventHandler(
		log.With(),
		s.mockEventListener,
		common.Address{},
		s.domainID,
		s.msgChan)
}

func (s *RetryEventHandlerTestSuite) Test_FetchRetryEventsFails() {
	s.mockEventListener.EXPECT().FetchRetryEvents(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]events.RetryEvent{}, fmt.Errorf("error"))

	err := s.retryEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))

	s.NotNil(err)
	s.Equal(len(s.msgChan), 0)
}

func (s *RetryEventHandlerTestSuite) Test_FetchRetryEvents_ValidRetry() {
	s.mockEventListener.EXPECT().FetchRetryEvents(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return([]events.RetryEvent{
		{SourceDomainID: 2, DestinationDomainID: 3, ResourceID: [32]byte{1}, BlockHeight: big.NewInt(100)},
		{SourceDomainID: 3, DestinationDomainID: 4, ResourceID: [32]byte{2}, BlockHeight: big.NewInt(101)},
	}, nil)

	err := s.retryEventHandler.HandleEvents(big.NewInt(0), big.NewInt(5))
	msgs2 := <-s.msgChan
	msgs1 := <-s.msgChan

	s.Nil(err)
	s.Equal(msgs1, []*message.Message{
		{
			Source:      s.domainID,
			Destination: 2,
			Data: retry.RetryMessageData{
				SourceDomainID: 2, DestinationDomainID: 3, ResourceID: [32]byte{1}, BlockHeight: big.NewInt(100),
			},
			ID:   "retry-2-3",
			Type: retry.RetryMessageType,
		},
	})
	s.Equal(msgs2, []*message.Message{
		{
			Source:      s.domainID,
			Destination: 3,
			Data: retry.RetryMessageData{
				SourceDomainID: 3, DestinationDomainID: 4, ResourceID: [32]byte{2}, BlockHeight: big.NewInt(101),
			},
			ID:   "retry-3-4",
			Type: retry.RetryMessageType,
		},
	})
}
