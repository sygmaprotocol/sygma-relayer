package bully

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

type BullyTestSuite struct {
	suite.Suite
	mockController        *gomock.Controller
	testHosts             []host.Host
	testCommunications    []communication.Communication
	testBullyCoordinators []Bully
	testProtocolID        protocol.ID
	testSessionID         string
}

const numberOfTestHosts = 3

func TestRunCommunicationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BullyTestSuite))
}

func (s *BullyTestSuite) SetupSuite() {
	s.testProtocolID = "test/protocol"
	s.testSessionID = "1"
}
func (s *BullyTestSuite) TearDownSuite() {}
func (s *BullyTestSuite) SetupTest() {
	s.mockController = gomock.NewController(s.T())

	// create test hosts
	for i := 0; i < numberOfTestHosts; i++ {
		privKeyForHost, _, _ := crypto.GenerateKeyPair(crypto.ECDSA, 1)
		newHost, _ := p2p.NewHost(privKeyForHost, relayer.MpcRelayerConfig{
			Peers: []*peer.AddrInfo{},
			Port:  uint16(4000 + i),
		})
		s.testHosts = append(s.testHosts, newHost)
		fmt.Printf("[%d] %s\n", i, newHost.ID().Pretty())
	}

	// populate peerstores
	peersAdrInfos := map[int][]*peer.AddrInfo{}
	for i := 0; i < numberOfTestHosts; i++ {
		for j := 0; j < numberOfTestHosts; j++ {
			if i != j {
				adrInfoForHost, _ := peer.AddrInfoFromString(fmt.Sprintf(
					"/ip4/127.0.0.1/tcp/%d/p2p/%s", 4000+j, s.testHosts[j].ID().Pretty(),
				))
				s.testHosts[i].Peerstore().AddAddr(adrInfoForHost.ID, adrInfoForHost.Addrs[0], peerstore.PermanentAddrTTL)
				peersAdrInfos[i] = append(peersAdrInfos[i], adrInfoForHost)
			}
		}
	}

	for i := 0; i < numberOfTestHosts; i++ {
		allowedPeers := peer.IDSlice{}
		for _, pInfo := range peersAdrInfos[i] {
			allowedPeers = append(allowedPeers, pInfo.ID)
		}

		com := p2p.NewCommunication(
			s.testHosts[i],
			s.testProtocolID,
			allowedPeers,
		)
		s.testCommunications = append(s.testCommunications, com)

		bcc := NewBullyCommunicationCoordinator(s.testHosts[i], s.testSessionID, relayer.BullyConfig{
			PingWaitTime:     2 * time.Second,
			PingBackOff:      10 * time.Second,
			PingInterval:     3 * time.Second,
			ElectionWaitTime: 3 * time.Second,
		})
		b := bcc.StartBullyCoordination(nil, s.testSessionID)
		s.testBullyCoordinators = append(s.testBullyCoordinators, b)
	}
}
func (s *BullyTestSuite) TearDownTest() {}

func (s *BullyTestSuite) TestCommunication_BroadcastMessage_SubscribersGotMessage() {

	coordinatorChan1 := make(chan peer.ID)
	errChan1 := make(chan error)
	go s.testBullyCoordinators[0].StartBully(coordinatorChan1, errChan1)

	coordinatorChan2 := make(chan peer.ID)
	errChan2 := make(chan error)
	go s.testBullyCoordinators[1].StartBully(coordinatorChan2, errChan2)

	coordinatorChan3 := make(chan peer.ID)
	errChan3 := make(chan error)
	go s.testBullyCoordinators[2].StartBully(coordinatorChan3, errChan3)

	time.Sleep(6 * time.Second)

	select {
	case c := <-coordinatorChan1:
		fmt.Printf("[1] %s\n", c.Pretty())
	case err := <-errChan1:
		fmt.Println(err)
	}

	select {
	case c := <-coordinatorChan2:
		fmt.Printf("[2] %s\n", c.Pretty())
	case err := <-errChan2:
		fmt.Println(err)
	}

	select {
	case c := <-coordinatorChan3:
		fmt.Printf("[3] %s\n", c.Pretty())
	case err := <-errChan3:
		fmt.Println(err)
	}

	//firstSubChannel := make(chan *communication.WrappedMessage)
	//s.testCommunications[0].Subscribe(s.testSessionID, communication.CoordinatorPingMsg, firstSubChannel)
	//
	//go func() {
	//	msg := <-firstSubChannel
	//	s.Equal("1", msg.SessionID)
	//	s.Equal(communication.CoordinatorPingMsg, msg.MessageType)
	//	s.Equal(s.testHosts[2].ID(), msg.From)
	//}()
	//
	//secondSubChannel := make(chan *communication.WrappedMessage)
	//s.testCommunications[1].Subscribe(s.testSessionID, communication.CoordinatorPingMsg, secondSubChannel)
	//
	//go func() {
	//	msg := <-secondSubChannel
	//	s.Equal(s.testSessionID, msg.SessionID)
	//	s.Equal(communication.CoordinatorPingMsg, msg.MessageType)
	//	s.Equal(s.testHosts[2].ID(), msg.From)
	//}()
	//
	//errChan := make(chan error)
	//
	//s.testCommunications[2].Broadcast(
	//	[]peer.ID{s.testHosts[0].ID(), s.testHosts[1].ID()},
	//	nil,
	//	communication.CoordinatorPingMsg,
	//	"1",
	//	errChan,
	//)
	//
	//s.Len(errChan, 0)
}
