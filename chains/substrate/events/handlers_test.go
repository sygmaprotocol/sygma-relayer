// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events_test

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	"github.com/centrifuge/go-substrate-rpc-client/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	mock_connection "github.com/ChainSafe/sygma-relayer/chains/substrate/events/mock"
)

type SystemUpdateHandlerTestSuite struct {
	suite.Suite
	conn                *mock_connection.MockChainConnection
	systemUpdateHandler *events.SystemUpdateEventHandler
}

func TestRunSystemUpdateHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SystemUpdateHandlerTestSuite))
}

func (s *SystemUpdateHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.conn = mock_connection.NewMockChainConnection(ctrl)
	s.systemUpdateHandler = events.NewSystemUpdateEventHandler(s.conn)
}

func (s *SystemUpdateHandlerTestSuite) Test_UpdateMetadataFails() {
	s.conn.EXPECT().UpdateMetatdata().Return(fmt.Errorf("error"))

	evtsRec := types.EventRecords{
		System_CodeUpdated: make([]types.EventSystemCodeUpdated, 1),
	}
	evts := events.Events{evtsRec}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(evts, msgChan)

	s.NotNil(err)
	s.Equal(len(msgChan), 0)
}

func (s *SystemUpdateHandlerTestSuite) Test_NoMetadataUpdate() {
	evts := events.Events{}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(evts, msgChan)

	s.Nil(err)
	s.Equal(len(msgChan), 0)
}

func (s *SystemUpdateHandlerTestSuite) Test_SuccesfullMetadataUpdate() {
	s.conn.EXPECT().UpdateMetatdata().Return(nil)

	evtsRec := types.EventRecords{
		System_CodeUpdated: make([]types.EventSystemCodeUpdated, 1),
	}
	evts := events.Events{evtsRec}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(evts, msgChan)

	s.Nil(err)
	s.Equal(len(msgChan), 0)
}
