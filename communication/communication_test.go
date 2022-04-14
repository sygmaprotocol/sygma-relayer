package communication_test

import (
	"fmt"
	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/ChainSafe/chainbridge-core/communication/p2p"
	"github.com/ChainSafe/chainbridge-core/config/relayer"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type CommunicationIntegrationTestSuite struct {
	suite.Suite
	mockController *gomock.Controller
	testHosts      []host.Host
	testProtocolID protocol.ID
	testSessionID  string
}

const numberOfTestHosts = 3

func TestRunCommunicationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CommunicationIntegrationTestSuite))
}

func (s *CommunicationIntegrationTestSuite) SetupSuite() {
	s.testProtocolID = "test/protocol"
	s.testSessionID = "1"
}
func (s *CommunicationIntegrationTestSuite) TearDownSuite() {}
func (s *CommunicationIntegrationTestSuite) SetupTest() {
	s.mockController = gomock.NewController(s.T())

	// create test hosts
	for i := 0; i < numberOfTestHosts; i++ {
		privKeyForHost, _, _ := crypto.GenerateKeyPair(crypto.ECDSA, 1)
		newHost, _ := p2p.NewHost(privKeyForHost, relayer.MpcRelayerConfig{
			Peers: []*peer.AddrInfo{},
			Port:  uint16(4000 + i),
		})
		s.testHosts = append(s.testHosts, newHost)
	}

	// populate peerstores
	for i := 0; i < numberOfTestHosts; i++ {
		for j := 0; j < numberOfTestHosts; j++ {
			if i != j {
				adrInfoForHost, _ := peer.AddrInfoFromString(fmt.Sprintf(
					"/ip4/127.0.0.1/tcp/4000/p2p/%s", s.testHosts[j].ID().Pretty(),
				))
				s.testHosts[i].Peerstore().AddAddr(adrInfoForHost.ID, adrInfoForHost.Addrs[0], peerstore.PermanentAddrTTL)
			}
		}
	}
}
func (s *CommunicationIntegrationTestSuite) TearDownTest() {}

func (s *CommunicationIntegrationTestSuite) TestCommunication_BroadcastMessage_SubscribersGotMessage() {
	var testCommunications []communication.Communication
	for i := 0; i < numberOfTestHosts; i++ {
		com := p2p.NewCommunication(s.testHosts[i], s.testProtocolID)
		testCommunications = append(testCommunications, com)
	}

	firstSubChannel := make(chan *communication.WrappedMessage)
	testCommunications[0].Subscribe(s.testSessionID, communication.CoordinatorPingMsg, firstSubChannel)

	go func() {
		msg := <-firstSubChannel
		s.Equal("1", msg.SessionID)
		s.Equal(communication.CoordinatorPingMsg, msg.MessageType)
		s.Equal(s.testHosts[2].ID(), msg.From)
	}()

	secondSubChannel := make(chan *communication.WrappedMessage)
	testCommunications[1].Subscribe(s.testSessionID, communication.CoordinatorPingMsg, secondSubChannel)

	go func() {
		msg := <-secondSubChannel
		s.Equal(s.testSessionID, msg.SessionID)
		s.Equal(communication.CoordinatorPingMsg, msg.MessageType)
		s.Equal(s.testHosts[2].ID(), msg.From)
	}()

	testCommunications[2].Broadcast(
		[]peer.ID{s.testHosts[0].ID(), s.testHosts[1].ID()},
		nil,
		communication.CoordinatorPingMsg,
		"1",
	)
}

func (s *CommunicationIntegrationTestSuite) TestCommunication_BroadcastMessage_StopReceivingMessagesAfterUnsubscribe() {
	var testCommunications []communication.Communication
	for i := 0; i < numberOfTestHosts; i++ {
		com := p2p.NewCommunication(s.testHosts[i], s.testProtocolID)
		testCommunications = append(testCommunications, com)
	}

	/** Both subscribers got a message **/

	firstSubChannel := make(chan *communication.WrappedMessage)
	firstSubID := testCommunications[0].Subscribe(s.testSessionID, communication.CoordinatorPingMsg, firstSubChannel)

	go func() {
		msg := <-firstSubChannel
		s.Equal("1", msg.SessionID)
		s.Equal(communication.CoordinatorPingMsg, msg.MessageType)
		s.Equal(s.testHosts[2].ID(), msg.From)
	}()

	secondSubChannel := make(chan *communication.WrappedMessage)
	testCommunications[1].Subscribe(s.testSessionID, communication.CoordinatorPingMsg, secondSubChannel)

	go func() {
		msg := <-secondSubChannel
		s.Equal(s.testSessionID, msg.SessionID)
		s.Equal(communication.CoordinatorPingMsg, msg.MessageType)
		s.Equal(s.testHosts[2].ID(), msg.From)
	}()

	testCommunications[2].Broadcast(
		[]peer.ID{s.testHosts[0].ID(), s.testHosts[1].ID()},
		nil,
		communication.CoordinatorPingMsg,
		"1",
	)

	/** After unsubscribe only one subscriber got a message **/

	testCommunications[0].UnSubscribe(firstSubID)

	testCommunications[2].Broadcast(
		[]peer.ID{s.testHosts[0].ID(), s.testHosts[1].ID()},
		nil,
		communication.CoordinatorPingMsg,
		"1",
	)

	time.Sleep(1 * time.Second)

	go func() {
		select {
		case _ = <-firstSubChannel:
			s.Fail("host[0] should be unsubscribed")
			break
		default:
			break
		}
	}()

	go func() {
		msg := <-secondSubChannel
		s.Equal(s.testSessionID, msg.SessionID)
		s.Equal(communication.CoordinatorPingMsg, msg.MessageType)
		s.Equal(s.testHosts[2].ID(), msg.From)
	}()
}
