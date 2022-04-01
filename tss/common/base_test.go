package common_test

import (
	"context"
	"errors"
	"testing"

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
	mockCommunication *mock_tss.MockCommunication
}

func TestRunBaseTssTestSuite(t *testing.T) {
	suite.Run(t, new(BaseTssTestSuite))
}
func (s *BaseTssTestSuite) SetupTest() {
	s.gomockController = gomock.NewController(s.T())
	s.mockMessage = mock_tss.NewMockMessage(s.gomockController)
	s.mockCommunication = mock_tss.NewMockCommunication(s.gomockController)
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
	s.mockMessage.EXPECT().WireBytes().Return([]byte{}, &tss.MessageRouting{}, errors.New("error"))

	go baseTss.ProcessOutboundMessages(context.Background(), outChn, common.KeyGenMsg)
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
	s.mockMessage.EXPECT().IsBroadcast().Return(false)
	s.mockMessage.EXPECT().GetTo().Return([]*tss.PartyID{common.CreatePartyID("invalid")})

	go baseTss.ProcessOutboundMessages(context.Background(), outChn, common.KeyGenMsg)
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
		MsgID:         "keygen",
		Communication: s.mockCommunication,
		ErrChn:        errChn,
	}
	s.mockMessage.EXPECT().WireBytes().Return([]byte{}, &tss.MessageRouting{
		IsBroadcast: true,
		From:        common.CreatePartyID("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54"),
	}, nil)
	s.mockMessage.EXPECT().IsBroadcast().Return(true)
	s.mockCommunication.EXPECT().Broadcast(baseTss.Peers, gomock.Any(), common.KeyGenMsg, "keygen")

	go baseTss.ProcessOutboundMessages(context.Background(), outChn, common.KeyGenMsg)
	outChn <- s.mockMessage

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
	}, errors.New("error")).AnyTimes().Times(0)

	ctx, cancel := context.WithCancel(context.Background())
	go baseTss.ProcessOutboundMessages(ctx, outChn, common.KeyGenMsg)

	cancel()
	outChn <- s.mockMessage

	s.Equal(len(errChn), 0)
}
