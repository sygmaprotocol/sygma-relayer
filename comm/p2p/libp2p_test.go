package p2p_test

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ChainSafe/sygma/comm"
	"github.com/ChainSafe/sygma/comm/p2p"
	mock_network "github.com/ChainSafe/sygma/comm/p2p/mock/stream"

	mock_host "github.com/ChainSafe/sygma/comm/p2p/mock/host"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
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
func (s *Libp2pCommunicationTestSuite) TearDownSuite() {}
func (s *Libp2pCommunicationTestSuite) SetupTest() {
	s.mockController = gomock.NewController(s.T())
	s.mockHost = mock_host.NewMockHost(s.mockController)
}
func (s *Libp2pCommunicationTestSuite) TearDownTest() {}

func (s *Libp2pCommunicationTestSuite) TestLibp2pCommunication_MessageProcessing_ValidMessage() {
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

	// mock for s.Conn().RemotePeer()
	mockConn := mock_network.NewMockConn(s.mockController)
	mockConn.EXPECT().RemotePeer().Return(s.allowedPeers[0])
	mockStream.EXPECT().Conn().Return(mockConn)

	// mock stream reading
	// on first call return header representing length of the message
	firstCall := mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		length := uint32(len(bytes))
		lengthBytes := make([]byte, p2p.LengthHeader)
		binary.LittleEndian.PutUint32(lengthBytes, length)

		copy(p[:], lengthBytes)
		return 4, nil
	})
	// on second call return message bytes
	secondCall := mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		copy(p[:], bytes)
		return len(bytes), nil
	})
	gomock.InOrder(firstCall, secondCall)

	messageFromStream, err := c.ProcessMessageFromStream(mockStream)

	s.Nil(err)
	s.NotNil(messageFromStream)

	s.Equal(s.allowedPeers[0], messageFromStream.From)
	s.Equal(testWrappedMsg.MessageType, messageFromStream.MessageType)
	s.Equal(testWrappedMsg.SessionID, messageFromStream.SessionID)
	s.Nil(messageFromStream.Payload)
}

func (s *Libp2pCommunicationTestSuite) TestLibp2pCommunication_MessageProcessing_FailOnReadingFromStream() {
	s.mockHost.EXPECT().ID().Return(s.allowedPeers[0])
	s.mockHost.EXPECT().SetStreamHandler(s.testProtocolID, gomock.Any()).Return()
	c := p2p.NewCommunication(s.mockHost, s.testProtocolID)

	mockStream := mock_network.NewMockStream(s.mockController)
	mockStream.EXPECT().Read(gomock.Any()).Times(1).Return(0, errors.New("error on reading from stream"))

	// mock for s.Conn().RemotePeer()
	mockConn := mock_network.NewMockConn(s.mockController)
	mockConn.EXPECT().RemotePeer().AnyTimes().Return(s.allowedPeers[0])
	mockStream.EXPECT().Conn().AnyTimes().Return(mockConn)

	messageFromStream, err := c.ProcessMessageFromStream(mockStream)

	s.Nil(messageFromStream)
	s.NotNil(err)
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

	// mock for s.Conn().RemotePeer()
	mockConn := mock_network.NewMockConn(s.mockController)
	mockConn.EXPECT().RemotePeer().AnyTimes().Return(s.allowedPeers[0])
	mockStream.EXPECT().Conn().AnyTimes().Return(mockConn)

	// mock stream reading
	// on first call return header representing length of the message
	firstCall := mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		length := uint32(len(bytes))
		lengthBytes := make([]byte, p2p.LengthHeader)
		binary.LittleEndian.PutUint32(lengthBytes, length)

		copy(p[:], lengthBytes)
		return 4, nil
	})
	// on second call return message bytes
	secondCall := mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		copy(p[:], bytes)
		return len(bytes), nil
	})
	gomock.InOrder(firstCall, secondCall)

	testSubChannelFirst := make(chan *comm.WrappedMessage)
	c.Subscribe("1", comm.CoordinatorPingMsg, testSubChannelFirst)

	testSubChannelSecond := make(chan *comm.WrappedMessage)
	c.Subscribe("1", comm.CoordinatorPingMsg, testSubChannelSecond)

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
}
