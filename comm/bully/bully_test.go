package bully

import (
	"fmt"
	"github.com/ChainSafe/chainbridge-core/comm"
	"github.com/ChainSafe/chainbridge-core/comm/p2p"
	"github.com/ChainSafe/chainbridge-core/config/relayer"
	"github.com/ChainSafe/chainbridge-core/tss/common"
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
	testCommunications    []comm.Communication
	testBullyCoordinators []*CommunicationCoordinator
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

	pirs := peer.IDSlice{}
	// create test hosts
	for i := 0; i < numberOfTestHosts; i++ {
		privKeyForHost, _, _ := crypto.GenerateKeyPair(crypto.ECDSA, 1)
		newHost, _ := p2p.NewHost(privKeyForHost, relayer.MpcRelayerConfig{
			Peers: []*peer.AddrInfo{},
			Port:  uint16(4000 + i),
		})
		s.testHosts = append(s.testHosts, newHost)
		fmt.Printf("[%d] %s\n", i, newHost.ID().Pretty())
		pirs = append(pirs, newHost.ID())
	}

	sortPeersForSession := common.SortPeersForSession(pirs, s.testSessionID)
	fmt.Println("---- SORTED ----")
	for i := range sortPeersForSession {
		fmt.Print(sortPeersForSession[i].ID.Pretty())
		if i == 0 {
			fmt.Print(" L\n")
		} else {
			fmt.Print("\n")
		}
	}
	fmt.Println("---- SORTED ----")

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

		bcc := NewCommunicationCoordinatorFactory(s.testHosts[i], relayer.BullyConfig{
			PingWaitTime:     2 * time.Second,
			PingBackOff:      10 * time.Second,
			PingInterval:     3 * time.Second,
			ElectionWaitTime: 3 * time.Second,
		})
		b := bcc.NewCommunicationCoordinator(nil, s.testSessionID)
		s.testBullyCoordinators = append(s.testBullyCoordinators, &b)
	}
}
func (s *BullyTestSuite) TearDownTest() {}

func (s *BullyTestSuite) TestCommunication_BroadcastMessage_SubscribersGotMessage() {
	time.Sleep(2 * time.Second)
	coordinatorChan1 := make(chan peer.ID)
	errChan1 := make(chan error)
	go s.testBullyCoordinators[0].StartBully(coordinatorChan1, errChan1)

	coordinatorChan2 := make(chan peer.ID)
	errChan2 := make(chan error)
	go s.testBullyCoordinators[1].StartBully(coordinatorChan2, errChan2)

	coordinatorChan3 := make(chan peer.ID)
	errChan3 := make(chan error)
	go s.testBullyCoordinators[2].StartBully(coordinatorChan3, errChan3)

	time.Sleep(10 * time.Second)

	fmt.Printf("[1] for %s\n", s.testHosts[0].ID().Pretty())
	fmt.Printf("[1] save %s\n", s.testBullyCoordinators[0].coordinator())

	fmt.Printf("[2] for %s\n", s.testHosts[1].ID().Pretty())
	fmt.Printf("[2] save %s\n", s.testBullyCoordinators[1].coordinator())

	fmt.Printf("[3] for %s\n", s.testHosts[2].ID().Pretty())
	fmt.Printf("[3] save %s\n", s.testBullyCoordinators[2].coordinator())

	//select {
	//case c := <-coordinatorChan1:
	//	fmt.Printf("[1] for %s\n", s.testHosts[0].ID().Pretty())
	//	fmt.Println("-----------------------------------------")
	//	fmt.Printf("[1] chan %s\n", c.Pretty())
	//	fmt.Printf("[1] save %s\n", s.testBullyCoordinators[0].coordinator())
	//	fmt.Println("-----------------------------------------")
	//case err := <-errChan1:
	//	fmt.Println(err)
	//}
	//
	//select {
	//case c := <-coordinatorChan2:
	//	fmt.Printf("[2] for %s\n", s.testHosts[1].ID().Pretty())
	//	fmt.Println("-----------------------------------------")
	//	fmt.Printf("[2] chan %s\n", c.Pretty())
	//	fmt.Printf("[2] save %s\n", s.testBullyCoordinators[1].coordinator())
	//	fmt.Println("-----------------------------------------")
	//case err := <-errChan2:
	//	fmt.Println(err)
	//}
	//
	//select {
	//case c := <-coordinatorChan3:
	//	fmt.Printf("[3] for %s\n", s.testHosts[2].ID().Pretty())
	//	fmt.Println("-----------------------------------------")
	//	fmt.Printf("[3] chan %s\n", c.Pretty())
	//	fmt.Printf("[3] save %s\n", s.testBullyCoordinators[2].coordinator())
	//	fmt.Println("-----------------------------------------")
	//case err := <-errChan3:
	//	fmt.Println(err)
	//}

}
