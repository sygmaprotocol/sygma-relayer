package p2p

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/ChainSafe/chainbridge-core/communication"
	mock_host "github.com/ChainSafe/chainbridge-core/communication/p2p/mock/host"
	mock_network "github.com/ChainSafe/chainbridge-core/communication/p2p/mock/stream"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"testing"
)

type Libp2pCommunicationTestSuite struct {
	suite.Suite
	mockController *gomock.Controller
	testPeer       peer.ID
	mockHost       *mock_host.MockHost
	testProtocolID protocol.ID
}

func TestRunLibp2pCommunicationTestSuite(t *testing.T) {
	suite.Run(t, new(Libp2pCommunicationTestSuite))
}

func (s *Libp2pCommunicationTestSuite) SetupSuite() {
	aInfo, _ := peer.AddrInfoFromString(
		"/ip4/127.0.0.1/tcp/4002/p2p/QmeWhpY8tknHS29gzf9TAsNEwfejTCNJ7vFpmkV6rNUgyq",
	)
	s.testPeer = aInfo.ID
	s.testProtocolID = "test/protocol"
}
func (s *Libp2pCommunicationTestSuite) TearDownSuite() {}
func (s *Libp2pCommunicationTestSuite) SetupTest() {
	s.mockController = gomock.NewController(s.T())
	s.mockHost = mock_host.NewMockHost(s.mockController)
}
func (s *Libp2pCommunicationTestSuite) TearDownTest() {}

func (s *Libp2pCommunicationTestSuite) TestLibp2pCommunication_MessageProcessing_ValidMessage() {
	c := Libp2pCommunication{
		SessionSubscriptionManager: NewSessionSubscriptionManager(),
		h:                          s.mockHost,
		protocolID:                 s.testProtocolID,
		streamManager:              NewStreamManager(),
		logger:                     zerolog.Logger{},
	}

	testWrappedMsg := communication.WrappedMessage{
		MessageType: communication.CoordinatorPingMsg,
		SessionID:   "1",
		Payload:     nil,
		From:        s.testPeer,
	}

	bytes, _ := json.Marshal(testWrappedMsg)

	mockStream := mock_network.NewMockStream(s.mockController)
	// on first call return header representing length of the message
	firstCall := mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		length := uint32(len(bytes))
		lengthBytes := make([]byte, LengthHeader)
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

	messageFromStream, err := c.processMessageFromStream(mockStream)

	s.Nil(err)
	s.NotNil(messageFromStream)

	s.Equal(testWrappedMsg.From, messageFromStream.From)
	s.Equal(testWrappedMsg.MessageType, messageFromStream.MessageType)
	s.Equal(testWrappedMsg.SessionID, messageFromStream.SessionID)
	s.Nil(messageFromStream.Payload)
}

func (s *Libp2pCommunicationTestSuite) TestLibp2pCommunication_MessageProcessing_FailOnReadingFromStream() {
	c := Libp2pCommunication{
		SessionSubscriptionManager: NewSessionSubscriptionManager(),
		h:                          s.mockHost,
		protocolID:                 s.testProtocolID,
		streamManager:              NewStreamManager(),
		logger:                     zerolog.Logger{},
	}

	mockStream := mock_network.NewMockStream(s.mockController)
	mockStream.EXPECT().Read(gomock.Any()).Times(1).Return(0, errors.New("error on reading from stream"))

	messageFromStream, err := c.processMessageFromStream(mockStream)

	s.Nil(messageFromStream)
	s.NotNil(err)
}

func (s *Libp2pCommunicationTestSuite) TestLib2pCommunication_MessageProcessing_FailOnUnmarshallingMessage() {
	c := Libp2pCommunication{
		SessionSubscriptionManager: NewSessionSubscriptionManager(),
		h:                          s.mockHost,
		protocolID:                 s.testProtocolID,
		streamManager:              NewStreamManager(),
		logger:                     zerolog.Logger{},
	}

	testWrappedMsg := communication.WrappedMessage{
		MessageType: communication.CoordinatorPingMsg,
		SessionID:   "1",
		Payload:     nil,
		From:        "8tknHS29g", // invalid peer ID
	}

	bytes, _ := json.Marshal(testWrappedMsg)

	mockStream := mock_network.NewMockStream(s.mockController)
	// on first call return header representing length of the message
	firstCall := mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		length := uint32(len(bytes))
		lengthBytes := make([]byte, LengthHeader)
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

	messageFromStream, err := c.processMessageFromStream(mockStream)

	s.Nil(messageFromStream)
	s.NotNil(err)
}

func (s *Libp2pCommunicationTestSuite) TestLibp2pCommunication_HandlingMessagesFromStream_ValidMessageWithSubscribers() {
	c := Libp2pCommunication{
		SessionSubscriptionManager: NewSessionSubscriptionManager(),
		h:                          s.mockHost,
		protocolID:                 s.testProtocolID,
		streamManager:              NewStreamManager(),
		logger:                     zerolog.Logger{},
	}

	testWrappedMsg := communication.WrappedMessage{
		MessageType: communication.CoordinatorPingMsg,
		SessionID:   "1",
		Payload:     nil,
		From:        s.testPeer,
	}

	bytes, _ := json.Marshal(testWrappedMsg)

	mockStream := mock_network.NewMockStream(s.mockController)
	// on first call return header representing length of the message
	firstCall := mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		length := uint32(len(bytes))
		lengthBytes := make([]byte, LengthHeader)
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

	testSubChannelFirst := make(chan *communication.WrappedMessage)
	c.Subscribe("1", communication.CoordinatorPingMsg, testSubChannelFirst)

	testSubChannelSecond := make(chan *communication.WrappedMessage)
	c.Subscribe("1", communication.CoordinatorPingMsg, testSubChannelSecond)

	go c.streamHandlerFunc(mockStream)

	subMsgFirst := <-testSubChannelFirst
	s.NotNil(subMsgFirst)
	s.Equal(testWrappedMsg.From, subMsgFirst.From)
	s.Equal(testWrappedMsg.MessageType, subMsgFirst.MessageType)
	s.Equal(testWrappedMsg.SessionID, subMsgFirst.SessionID)
	s.Nil(subMsgFirst.Payload)

	subMsgSecond := <-testSubChannelSecond
	s.NotNil(subMsgSecond)
	s.Equal(testWrappedMsg.From, subMsgSecond.From)
	s.Equal(testWrappedMsg.MessageType, subMsgSecond.MessageType)
	s.Equal(testWrappedMsg.SessionID, subMsgSecond.SessionID)
	s.Nil(subMsgSecond.Payload)
}
