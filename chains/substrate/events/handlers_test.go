// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events_test

import (
	"fmt"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/types"
	mock_events "github.com/ChainSafe/sygma-relayer/chains/substrate/events/mock"

	"testing"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	substrate_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type SystemUpdateHandlerTestSuite struct {
	suite.Suite
	conn                *mock_events.MockChainConnection
	systemUpdateHandler *events.SystemUpdateEventHandler
}

func TestRunSystemUpdateHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SystemUpdateHandlerTestSuite))
}

func (s *SystemUpdateHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.conn = mock_events.NewMockChainConnection(ctrl)
	s.systemUpdateHandler = events.NewSystemUpdateEventHandler(s.conn)
}

func (s *SystemUpdateHandlerTestSuite) Test_UpdateMetadataFails() {
	s.conn.EXPECT().UpdateMetatdata().Return(fmt.Errorf("error"))

	evtsRec := substrate_types.EventRecords{
		System_CodeUpdated: make([]substrate_types.EventSystemCodeUpdated, 1),
	}
	evts := []*events.Events{{
		evtsRec,
		[]events.Deposit{},
	}}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(evts, msgChan)

	s.NotNil(err)
	s.Equal(len(msgChan), 0)
}

func (s *SystemUpdateHandlerTestSuite) Test_NoEvents() {
	evts := []*events.Events{}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(evts, msgChan)

	s.Nil(err)
	s.Equal(len(msgChan), 0)
}

func (s *SystemUpdateHandlerTestSuite) Test_NoMetadatUpdate() {
	evts := []*events.Events{{}}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(evts, msgChan)

	s.Nil(err)
	s.Equal(len(msgChan), 0)
}

func (s *SystemUpdateHandlerTestSuite) Test_SuccesfullMetadataUpdate() {
	s.conn.EXPECT().UpdateMetatdata().Return(nil)

	evtsRec := substrate_types.EventRecords{
		System_CodeUpdated: make([]substrate_types.EventSystemCodeUpdated, 1),
	}
	evts := []*events.Events{
		{},
		{
			EventRecords: evtsRec,
		},
	}
	msgChan := make(chan []*message.Message, 1)
	err := s.systemUpdateHandler.HandleEvents(evts, msgChan)

	s.Nil(err)
	s.Equal(len(msgChan), 0)
}

type DepositHandlerTestSuite struct {
	suite.Suite
	depositEventHandler *events.FungibleTransferEventHandler
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
	s.depositEventHandler = events.NewFungibleTransferEventHandler(s.domainID, s.mockDepositHandler)
}

func (s *DepositHandlerTestSuite) Test_HandleDepositFails_ExecutionContinue() {
	d1 := &events.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		DepositType:         message.FungibleTransfer,
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := &events.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		DepositType:         message.FungibleTransfer,
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.DepositType,
		d1.HandlerResponse,
	).Return(&message.Message{}, fmt.Errorf("error"))

	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d1.DepositType,
		d2.HandlerResponse,
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	msgChan := make(chan []*message.Message, 2)
	evtsRec := substrate_types.EventRecords{
		System_CodeUpdated: make([]substrate_types.EventSystemCodeUpdated, 1),
	}
	evts := []*events.Events{
		{
			evtsRec,
			[]events.Deposit{
				*d1, *d2,
			},
		},
	}
	err := s.depositEventHandler.HandleEvents(evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 2}})
}

func (s *DepositHandlerTestSuite) Test_SuccessfulHandleDeposit() {
	d1 := &events.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		DepositType:         message.FungibleTransfer,
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := &events.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		DepositType:         message.FungibleTransfer,
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.DepositType,
		d1.HandlerResponse,
	).Return(
		&message.Message{DepositNonce: 1},
		nil,
	)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.DepositType,
		d2.HandlerResponse,
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	msgChan := make(chan []*message.Message, 2)
	evtsRec := substrate_types.EventRecords{
		System_CodeUpdated: make([]substrate_types.EventSystemCodeUpdated, 1),
	}
	evts := []*events.Events{
		{
			evtsRec,
			[]events.Deposit{
				*d1, *d2,
			},
		},
	}
	err := s.depositEventHandler.HandleEvents(evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 1}, {DepositNonce: 2}})
}

func (s *DepositHandlerTestSuite) Test_HandleDepositPanics_ExecutionContinues() {
	d1 := &events.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		DepositType:         message.FungibleTransfer,
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := &events.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		DepositType:         message.FungibleTransfer,
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.DepositType,
		d1.HandlerResponse,
	).Do(func(sourceID, destID, nonce, resourceID, calldata, depositType, handlerResponse interface{}) {
		panic("error")
	})
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.DepositType,
		d2.HandlerResponse,
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)

	msgChan := make(chan []*message.Message, 2)
	evtsRec := substrate_types.EventRecords{
		System_CodeUpdated: make([]substrate_types.EventSystemCodeUpdated, 1),
	}
	evts := []*events.Events{
		{
			evtsRec,
			[]events.Deposit{
				*d1, *d2,
			},
		},
	}
	err := s.depositEventHandler.HandleEvents(evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 2}})
}

func (s *DepositHandlerTestSuite) Test_SuccessfulHandleDeposit_MultipleBlocks() {
	d1 := &events.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		DepositType:         message.FungibleTransfer,
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d2 := &events.Deposit{
		DepositNonce:        2,
		DestinationDomainID: 2,
		ResourceID:          types.ResourceID{},
		DepositType:         message.FungibleTransfer,
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	d3 := &events.Deposit{
		DepositNonce:        3,
		DestinationDomainID: 3,
		ResourceID:          types.ResourceID{},
		DepositType:         message.FungibleTransfer,
		HandlerResponse:     []byte{},
		Data:                []byte{},
	}
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d1.DestinationDomainID,
		d1.DepositNonce,
		d1.ResourceID,
		d1.Data,
		d1.DepositType,
		d1.HandlerResponse,
	).Return(
		&message.Message{DepositNonce: 1},
		nil,
	)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d2.DestinationDomainID,
		d2.DepositNonce,
		d2.ResourceID,
		d2.Data,
		d2.DepositType,
		d2.HandlerResponse,
	).Return(
		&message.Message{DepositNonce: 2},
		nil,
	)
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		d3.DestinationDomainID,
		d3.DepositNonce,
		d3.ResourceID,
		d3.Data,
		d3.DepositType,
		d3.HandlerResponse,
	).Return(
		&message.Message{DepositNonce: 3},
		nil,
	)

	msgChan := make(chan []*message.Message, 3)
	evtsRec := substrate_types.EventRecords{
		System_CodeUpdated: make([]substrate_types.EventSystemCodeUpdated, 1),
	}
	evts := []*events.Events{
		{
			evtsRec,
			[]events.Deposit{
				*d1, *d2,
			},
		},
		{
			evtsRec,
			[]events.Deposit{},
		},
		{
			evtsRec,
			[]events.Deposit{
				*d3,
			},
		},
	}
	err := s.depositEventHandler.HandleEvents(evts, msgChan)
	msgs := <-msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{DepositNonce: 1}, {DepositNonce: 2}, {DepositNonce: 3}})
}
