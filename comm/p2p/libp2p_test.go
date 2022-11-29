// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package p2p_test

import (
	"encoding/json"
	"fmt"
	"testing"

	comm "github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/comm/p2p"
	mock_host "github.com/ChainSafe/sygma-relayer/comm/p2p/mock/host"
	mock_network "github.com/ChainSafe/sygma-relayer/comm/p2p/mock/stream"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/suite"
)

type Libp2pCommunicationTestSuite struct {
	suite.Suite
	mockController *gomock.Controller
	mockHost       *mock_host.MockHost
	testProtocolID protocol.ID
	allowedPeers   peer.IDSlice
}

func TestRunLibp2pCommunicationTestSuite(t *testing.T) {
	suite.Run(t, new(Libp2pCommunicationTestSuite))
}

func (s *Libp2pCommunicationTestSuite) SetupSuite() {
	pid, _ := peer.Decode("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")
	s.allowedPeers = []peer.ID{pid}
	s.testProtocolID = "test/protocol"
}
func (s *Libp2pCommunicationTestSuite) SetupTest() {
	s.mockController = gomock.NewController(s.T())
	s.mockHost = mock_host.NewMockHost(s.mockController)
}

func (s *Libp2pCommunicationTestSuite) TestLibp2pCommunication_MessageProcessing_ValidMessage() {
	s.mockHost.EXPECT().ID().Return(s.allowedPeers[0])
	s.mockHost.EXPECT().SetStreamHandler(s.testProtocolID, gomock.Any()).Return()
	c := p2p.NewCommunication(s.mockHost, s.testProtocolID)

	msgChannel := make(chan *comm.WrappedMessage)
	c.Subscribe("1", comm.CoordinatorPingMsg, msgChannel)

	testWrappedMsg := comm.WrappedMessage{
		MessageType: comm.CoordinatorPingMsg,
		SessionID:   "1",
		Payload:     nil,
	}
	bytes, _ := json.Marshal(testWrappedMsg)

	mockStream := mock_network.NewMockStream(s.mockController)
	mockConn := mock_network.NewMockConn(s.mockController)
	mockConn.EXPECT().RemotePeer().Return(s.allowedPeers[0])
	mockStream.EXPECT().Conn().Return(mockConn)

	firstCall := mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		copy(p[:], []byte(fmt.Sprintf("%s \n", string(bytes[:]))))
		return len(bytes), nil
	})
	secondCall := mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		copy(p[:], []byte("\n"))
		return len(bytes), nil
	})
	gomock.InOrder(firstCall, secondCall)

	c.ProcessMessagesFromStream(mockStream)

	msg := <-msgChannel

	s.Equal(s.allowedPeers[0], msg.From)
	s.Equal(testWrappedMsg.MessageType, msg.MessageType)
	s.Equal(testWrappedMsg.SessionID, msg.SessionID)
	s.Nil(msg.Payload)
}

func (s *Libp2pCommunicationTestSuite) TestLibp2pCommunication_StreamHandlerFunction_ValidMessageWithSubscribers() {
	s.mockHost.EXPECT().ID().Return(s.allowedPeers[0])
	s.mockHost.EXPECT().SetStreamHandler(s.testProtocolID, gomock.Any()).Return()
	c := p2p.NewCommunication(s.mockHost, s.testProtocolID)

	testWrappedMsg := comm.WrappedMessage{
		MessageType: comm.CoordinatorPingMsg,
		SessionID:   "1",
		Payload:     nil,
	}

	bytes, _ := json.Marshal(testWrappedMsg)

	mockStream := mock_network.NewMockStream(s.mockController)
	mockConn := mock_network.NewMockConn(s.mockController)
	mockConn.EXPECT().RemotePeer().AnyTimes().Return(s.allowedPeers[0])
	mockStream.EXPECT().Conn().AnyTimes().Return(mockConn)
	mockStream.EXPECT().Close()

	firstCall := mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		copy(p[:], []byte(fmt.Sprintf("%s \n", string(bytes[:]))))
		return len(bytes), nil
	})
	secondCall := mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		copy(p[:], []byte("\n"))
		return len(bytes), nil
	})
	gomock.InOrder(firstCall, secondCall)

	testSubChannelFirst := make(chan *comm.WrappedMessage)
	subID1 := c.Subscribe("1", comm.CoordinatorPingMsg, testSubChannelFirst)

	testSubChannelSecond := make(chan *comm.WrappedMessage)
	subID2 := c.Subscribe("1", comm.CoordinatorPingMsg, testSubChannelSecond)

	go c.StreamHandlerFunc(mockStream)

	subMsgFirst := <-testSubChannelFirst
	s.NotNil(subMsgFirst)
	s.Equal(s.allowedPeers[0], subMsgFirst.From)
	s.Equal(testWrappedMsg.MessageType, subMsgFirst.MessageType)
	s.Equal(testWrappedMsg.SessionID, subMsgFirst.SessionID)
	s.Nil(subMsgFirst.Payload)

	subMsgSecond := <-testSubChannelSecond
	s.NotNil(subMsgSecond)
	s.Equal(s.allowedPeers[0], subMsgSecond.From)
	s.Equal(testWrappedMsg.MessageType, subMsgSecond.MessageType)
	s.Equal(testWrappedMsg.SessionID, subMsgSecond.SessionID)
	s.Nil(subMsgSecond.Payload)

	c.UnSubscribe(subID1)
	c.UnSubscribe(subID2)
}
