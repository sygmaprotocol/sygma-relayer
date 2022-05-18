package bully

import (
	"fmt"
	"github.com/ChainSafe/chainbridge-core/comm"
	"github.com/ChainSafe/chainbridge-core/comm/p2p"
	"github.com/ChainSafe/chainbridge-core/comm/static"
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
	allowedPeers          peer.IDSlice
}

const numberOfTestHosts = 3

func TestRunCommunicationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BullyTestSuite))
}

func (s *BullyTestSuite) SetupSuite() {
	s.testProtocolID = "test/protocol"
	s.testSessionID = "1"

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
		s.allowedPeers = allowedPeers
	}
}
func (s *BullyTestSuite) TearDownSuite() {}
func (s *BullyTestSuite) SetupTest() {
	s.mockController = gomock.NewController(s.T())

	for i := 0; i < numberOfTestHosts; i++ {

		com := p2p.NewCommunication(
			s.testHosts[i],
			s.testProtocolID,
			s.allowedPeers,
		)
		s.testCommunications = append(s.testCommunications, com)

		bcc := NewCommunicationCoordinatorFactory(s.testHosts[i], relayer.BullyConfig{
			PingWaitTime:     2 * time.Second,
			PingBackOff:      10 * time.Second,
			PingInterval:     3 * time.Second,
			ElectionWaitTime: 3 * time.Second,
			BullyWaitTime:    10 * time.Second,
		})
		b := bcc.NewCommunicationCoordinator(s.testSessionID)
		s.testBullyCoordinators = append(s.testBullyCoordinators, b)
	}
}
func (s *BullyTestSuite) TearDownTest() {}

func (s *BullyTestSuite) TestBully_GetCoordinator_AllStartAtSameTime() {
	time.Sleep(3 * time.Second)

	resultChan := make(chan peer.ID)

	cc := static.NewStaticCommunicationCoordinator(s.testHosts[0])
	coordinator, _ := cc.GetCoordinator(s.testSessionID)

	go func() {
		c, err := s.testBullyCoordinators[0].GetCoordinator(nil)
		s.Nil(err)
		resultChan <- c
	}()

	go func() {
		c, err := s.testBullyCoordinators[1].GetCoordinator(nil)
		s.Nil(err)
		resultChan <- c
	}()

	go func() {
		c, err := s.testBullyCoordinators[2].GetCoordinator(nil)
		s.Nil(err)
		resultChan <- c
	}()

	for i := 0; i < 3; i++ {
		select {
		case c := <-resultChan:
			s.Equal(coordinator, c)
		}
	}
}

type Tmp struct {
	ID    peer.ID
	rName string
}

func (s *BullyTestSuite) TestBully_GetCoordinator_OneDelay() {
	time.Sleep(3 * time.Second)

	resultChan := make(chan Tmp)

	cc := static.NewStaticCommunicationCoordinator(s.testHosts[0])
	coordinator, _ := cc.GetCoordinator(s.testSessionID)

	go func() {
		time.Sleep(1 * time.Second)
		c, err := s.testBullyCoordinators[0].GetCoordinator(nil)
		s.Nil(err)
		resultChan <- Tmp{
			ID:    c,
			rName: "R1",
		}
	}()

	go func() {
		time.Sleep(2 * time.Second)
		c, err := s.testBullyCoordinators[1].GetCoordinator(nil)
		s.Nil(err)
		resultChan <- Tmp{
			ID:    c,
			rName: "R2",
		}
	}()

	go func() {
		c, err := s.testBullyCoordinators[2].GetCoordinator(nil)
		s.Nil(err)
		resultChan <- Tmp{
			ID:    c,
			rName: "R3",
		}
	}()

	for i := 0; i < 3; i++ {
		select {
		case c := <-resultChan:
			fmt.Printf("[%s] %s\n", c.rName, c.ID.Pretty())
			s.Equal(coordinator, c.ID)
		}
	}
}
