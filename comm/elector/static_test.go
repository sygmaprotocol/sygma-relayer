package elector

import (
	"context"
	"fmt"
	"testing"

	"github.com/ChainSafe/chainbridge-core/comm/p2p"
	"github.com/ChainSafe/chainbridge-core/config/relayer"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/stretchr/testify/suite"
)

type CoordinatorElectorTestSuite struct {
	suite.Suite
	mockController *gomock.Controller
	testHosts      []host.Host
	testPeers      peer.IDSlice
}

const numberOfTestHosts = 3

func TestRunStaticCommunicationCoordinatorTestSuite(t *testing.T) {
	suite.Run(t, new(CoordinatorElectorTestSuite))
}

func (s *CoordinatorElectorTestSuite) SetupSuite()    {}
func (s *CoordinatorElectorTestSuite) TearDownSuite() {}
func (s *CoordinatorElectorTestSuite) SetupTest() {
	s.mockController = gomock.NewController(s.T())

	peers := peer.IDSlice{}
	// create test hosts
	for i := 0; i < numberOfTestHosts; i++ {
		privKeyForHost, _, _ := crypto.GenerateKeyPair(crypto.ECDSA, 1)
		newHost, _ := p2p.NewHost(privKeyForHost, relayer.MpcRelayerConfig{
			Peers: []*peer.AddrInfo{},
			Port:  uint16(4000 + i),
		})
		s.testHosts = append(s.testHosts, newHost)
		peers = append(peers, newHost.ID())
	}
	s.testPeers = peers

	// populate peerstores
	peersAdrInfos := map[int][]*peer.AddrInfo{}
	for i := 0; i < numberOfTestHosts; i++ {
		for j := 0; j < numberOfTestHosts; j++ {
			if i != j {
				adrInfoForHost, _ := peer.AddrInfoFromString(fmt.Sprintf(
					"/ip4/127.0.0.1/tcp/%d/p2p/%s", 4000+j, s.testHosts[j].ID().Pretty(),
				))
				s.testHosts[i].Peerstore().AddAddr(
					adrInfoForHost.ID, adrInfoForHost.Addrs[0], peerstore.PermanentAddrTTL,
				)
				peersAdrInfos[i] = append(peersAdrInfos[i], adrInfoForHost)
			}
		}
	}
}
func (s *CoordinatorElectorTestSuite) TearDownTest() {}

func (s *CoordinatorElectorTestSuite) TestStaticCommunicationCoordinator_GetCoordinator_Success() {
	staticCommunicationCoordinator := NewCoordinatorElector("1")

	coordinator1, err := staticCommunicationCoordinator.Coordinator(context.Background(), s.testPeers)
	s.Nil(err)
	s.NotNil(coordinator1)
	s.Contains(s.testPeers, coordinator1)
}
