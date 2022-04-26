package common_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ChainSafe/chainbridge-core/communication"
	mock_communication "github.com/ChainSafe/chainbridge-core/communication/mock"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	mock_tss "github.com/ChainSafe/chainbridge-core/tss/common/mock"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/suite"
)

type BaseTssTestSuite struct {
	suite.Suite
	gomockController  *gomock.Controller
	mockMessage       *mock_tss.MockMessage
	mockCommunication *mock_communication.MockCommunication
	mockParty         *mock_tss.MockParty
}

func TestRunBaseTssTestSuite(t *testing.T) {
	suite.Run(t, new(BaseTssTestSuite))
}

func (s *BaseTssTestSuite) SetupTest() {
	s.gomockController = gomock.NewController(s.T())
	s.mockMessage = mock_tss.NewMockMessage(s.gomockController)
	s.mockParty = mock_tss.NewMockParty(s.gomockController)
	s.mockCommunication = mock_communication.NewMockCommunication(s.gomockController)
}

func (s *BaseTssTestSuite) Test_BroadcastPeers_BroadcastMessage() {
	peerID1, _ := peer.Decode("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")
	peerID2, _ := peer.Decode("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")
	baseTss := common.BaseTss{
		Peers: []peer.ID{peerID1, peerID2},
	}
	s.mockMessage.EXPECT().IsBroadcast().Return(true)

	broadcastPeers, err := baseTss.BroadcastPeers(s.mockMessage)

	s.Nil(err)
	s.Equal(broadcastPeers, []peer.ID{peerID1, peerID2})
}

func (s *BaseTssTestSuite) Test_BroadcastPeers_TargetedMessage() {
	peerID1, _ := peer.Decode("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")
	peerID2, _ := peer.Decode("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")
	baseTss := common.BaseTss{
		Peers: []peer.ID{peerID1, peerID2},
	}
	s.mockMessage.EXPECT().IsBroadcast().Return(false)
	s.mockMessage.EXPECT().GetTo().Return([]*tss.PartyID{common.CreatePartyID("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")})

	broadcastPeers, err := baseTss.BroadcastPeers(s.mockMessage)

	s.Nil(err)
	s.Equal(broadcastPeers, []peer.ID{peerID1})
}

func (s *BaseTssTestSuite) Test_PopulatePartyStore() {
	baseTss := common.BaseTss{
		PartyStore: make(map[string]*tss.PartyID),
	}
	peerID1 := "QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR"
	peerID2 := "QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54"
	parties := []*tss.PartyID{common.CreatePartyID(peerID1), common.CreatePartyID(peerID2)}

	baseTss.PopulatePartyStore(parties)

	s.Equal(baseTss.PartyStore[peerID1], common.CreatePartyID(peerID1))
}

func (s *BaseTssTestSuite) Test_ProcessOutboundMessages_InvalidWireBytes() {
	outChn := make(chan tss.Message)
	errChn := make(chan error)
	baseTss := common.BaseTss{
		ErrChn: errChn,
	}
	s.mockMessage.EXPECT().String().Return("MSG")
	s.mockMessage.EXPECT().WireBytes().Return([]byte{}, &tss.MessageRouting{}, errors.New("error"))

	go baseTss.ProcessOutboundMessages(context.Background(), outChn, communication.TssKeyGenMsg)
	outChn <- s.mockMessage
	err := <-errChn

	s.NotNil(err)
}

func (s *BaseTssTestSuite) Test_ProcessOutboundMessages_InvalidBroadcastPeers() {
	outChn := make(chan tss.Message)
	errChn := make(chan error)
	baseTss := common.BaseTss{
		ErrChn: errChn,
	}
	s.mockMessage.EXPECT().WireBytes().Return([]byte{}, &tss.MessageRouting{
		IsBroadcast: true,
		From:        common.CreatePartyID("invalid"),
	}, nil)
	s.mockMessage.EXPECT().String().Return("MSG")
	s.mockMessage.EXPECT().IsBroadcast().Return(false)
	s.mockMessage.EXPECT().GetTo().Return([]*tss.PartyID{common.CreatePartyID("invalid")})

	go baseTss.ProcessOutboundMessages(context.Background(), outChn, communication.TssKeyGenMsg)
	time.Sleep(time.Millisecond * 50)
	outChn <- s.mockMessage
	err := <-errChn

	s.NotNil(err)
}

func (s *BaseTssTestSuite) Test_ProcessOutboundMessages_ValidMessage() {
	outChn := make(chan tss.Message)
	errChn := make(chan error, 1)
	peerID, _ := peer.Decode("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")
	baseTss := common.BaseTss{
		Peers:         []peer.ID{peerID},
		SID:           "keygen",
		Communication: s.mockCommunication,
		ErrChn:        errChn,
	}
	s.mockMessage.EXPECT().String().Return("MSG")
	s.mockMessage.EXPECT().WireBytes().Return([]byte{}, &tss.MessageRouting{
		IsBroadcast: true,
		From:        common.CreatePartyID("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54"),
	}, nil)
	s.mockMessage.EXPECT().IsBroadcast().Return(true)
	s.mockCommunication.EXPECT().Broadcast(baseTss.Peers, gomock.Any(), communication.TssKeyGenMsg, "keygen", gomock.Any())

	go baseTss.ProcessOutboundMessages(context.Background(), outChn, communication.TssKeyGenMsg)
	time.Sleep(time.Millisecond * 50)
	outChn <- s.mockMessage
	time.Sleep(time.Millisecond * 50)

	s.Equal(len(errChn), 0)
}

func (s *BaseTssTestSuite) Test_ProcessOutboundMessages_ContextCanceled() {
	outChn := make(chan tss.Message, 1)
	errChn := make(chan error, 1)
	peerID, _ := peer.Decode("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")
	baseTss := common.BaseTss{
		Peers:  []peer.ID{peerID},
		ErrChn: errChn,
	}
	s.mockMessage.EXPECT().WireBytes().Return([]byte{}, &tss.MessageRouting{
		IsBroadcast: true,
		From:        common.CreatePartyID("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54"),
	}, errors.New("error")).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	go baseTss.ProcessOutboundMessages(ctx, outChn, communication.TssKeyGenMsg)

	cancel()
	time.Sleep(time.Millisecond * 10)
	outChn <- s.mockMessage

	s.Equal(len(errChn), 0)
}

func (s *BaseTssTestSuite) Test_ProcessInboundMessages_InvalidMessage() {
	msgChan := make(chan *communication.WrappedMessage)
	errChn := make(chan error, 1)
	partyStore := make(map[string]*tss.PartyID)
	peerID := "QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54"
	party := common.CreatePartyID(peerID)
	partyStore[peerID] = party
	baseTss := common.BaseTss{
		ErrChn:     errChn,
		PartyStore: partyStore,
		Party:      s.mockParty,
	}
	msg, _ := common.MarshalTssMessage([]byte{1}, true, peerID)
	wrappedMsg := &communication.WrappedMessage{
		Payload: msg,
	}
	s.mockParty.EXPECT().UpdateFromBytes([]byte{1}, baseTss.PartyStore[peerID], true).Return(false, &tss.Error{})

	go baseTss.ProcessInboundMessages(context.Background(), msgChan)

	msgChan <- wrappedMsg
	err := <-errChn

	s.NotNil(err)
}

func (s *BaseTssTestSuite) Test_ProcessInboundMessages_ValidMessage() {
	msgChan := make(chan *communication.WrappedMessage)
	errChn := make(chan error, 1)
	partyStore := make(map[string]*tss.PartyID)
	peerID := "QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54"
	party := common.CreatePartyID(peerID)
	partyStore[peerID] = party
	baseTss := common.BaseTss{
		ErrChn:     errChn,
		PartyStore: partyStore,
		Party:      s.mockParty,
	}
	msg, _ := common.MarshalTssMessage([]byte{1}, true, peerID)
	wrappedMsg := &communication.WrappedMessage{
		Payload: msg,
	}
	s.mockParty.EXPECT().UpdateFromBytes([]byte{1}, baseTss.PartyStore[peerID], true).Return(true, nil)

	go baseTss.ProcessInboundMessages(context.Background(), msgChan)

	msgChan <- wrappedMsg

	s.Equal(len(errChn), 0)
}

func (s *BaseTssTestSuite) Test_ProcessInboundMessages_ContextCanceled() {
	msgChan := make(chan *communication.WrappedMessage, 1)
	errChn := make(chan error, 1)
	partyStore := make(map[string]*tss.PartyID)
	peerID := "QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54"
	party := common.CreatePartyID(peerID)
	partyStore[peerID] = party
	baseTss := common.BaseTss{
		ErrChn:     errChn,
		PartyStore: partyStore,
		Party:      s.mockParty,
	}
	msg, _ := common.MarshalTssMessage([]byte{1}, true, peerID)
	wrappedMsg := &communication.WrappedMessage{
		Payload: msg,
	}
	s.mockParty.EXPECT().UpdateFromBytes([]byte{1}, baseTss.PartyStore[peerID], true).Return(false, &tss.Error{}).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	go baseTss.ProcessInboundMessages(ctx, msgChan)

	cancel()
	msgChan <- wrappedMsg

	s.Equal(len(errChn), 0)
}
