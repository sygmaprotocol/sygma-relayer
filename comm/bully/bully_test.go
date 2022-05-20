package bully

import (
	"fmt"
	"github.com/ChainSafe/chainbridge-core/comm"
	"github.com/ChainSafe/chainbridge-core/comm/p2p"
	"github.com/ChainSafe/chainbridge-core/comm/static"
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
	testCommunications    []comm.Communication
	testBullyCoordinators []*CommunicationCoordinator
	testProtocolID        protocol.ID
	testSessionID         string
}

type RelayerTestDescriber struct {
	name         string
	index        int
	isActive     bool
	initialDelay time.Duration
}

type BullyTestCase struct {
	name         string
	testRelayers []RelayerTestDescriber
}

func TestRunCommunicationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BullyTestSuite))
}

func (s *BullyTestSuite) SetupSuite() {
	s.testProtocolID = "test/protocol"
	s.testSessionID = "1"
}
func (s *BullyTestSuite) TearDownSuite() {}
func (s *BullyTestSuite) SetupTest()     {}

func (s *BullyTestSuite) SetupIndividualTest(c BullyTestCase) map[peer.ID]string {
	s.mockController = gomock.NewController(s.T())
	s.testHosts = []host.Host{}
	s.testCommunications = []comm.Communication{}
	s.testBullyCoordinators = []*CommunicationCoordinator{}

	numberOfTestHosts := len(c.testRelayers)

	allowedPeers := peer.IDSlice{}
	// create test hosts
	for i := 0; i < numberOfTestHosts; i++ {
		privKeyForHost, _, _ := crypto.GenerateKeyPair(crypto.ECDSA, 1)
		newHost, _ := p2p.NewHost(privKeyForHost, relayer.MpcRelayerConfig{
			Peers: []*peer.AddrInfo{},
			Port:  uint16(4000 + i),
		})
		s.testHosts = append(s.testHosts, newHost)
		allowedPeers = append(allowedPeers, newHost.ID())
	}

	// populate peerstores
	for i := 0; i < numberOfTestHosts; i++ {
		for j := 0; j < numberOfTestHosts; j++ {
			if i != j {
				adrInfoForHost, _ := peer.AddrInfoFromString(fmt.Sprintf(
					"/ip4/127.0.0.1/tcp/%d/p2p/%s", 4000+j, s.testHosts[j].ID().Pretty(),
				))
				s.testHosts[i].Peerstore().AddAddr(adrInfoForHost.ID, adrInfoForHost.Addrs[0], peerstore.PermanentAddrTTL)
			}
		}
	}

	names := map[peer.ID]string{}
	for i := 0; i < numberOfTestHosts; i++ {
		names[s.testHosts[i].ID()] = fmt.Sprintf("R%d", i)

		if c.testRelayers[i].isActive {
			com := p2p.NewCommunication(
				s.testHosts[i],
				s.testProtocolID,
				allowedPeers,
			)
			s.testCommunications = append(s.testCommunications, com)

			bcc := NewCommunicationCoordinatorFactory(s.testHosts[i], relayer.BullyConfig{
				PingWaitTime:     1 * time.Second,
				PingBackOff:      1 * time.Second,
				PingInterval:     1 * time.Second,
				ElectionWaitTime: 2 * time.Second,
				BullyWaitTime:    10 * time.Second,
			})

			b := bcc.NewCommunicationCoordinator(s.testSessionID, names)
			s.testBullyCoordinators = append(s.testBullyCoordinators, b)
		}
	}
	return names
}
func (s *BullyTestSuite) TearDownTest() {}

func (s *BullyTestSuite) TestBully_GetCoordinator_OneDelay() {

	testCases := []BullyTestCase{
		//{
		//	name: "basic test",
		//	testRelayers: []RelayerTestDescriber{
		//		{
		//			name:         "R1",
		//			index:        0,
		//			isActive:     true,
		//			initialDelay: 0,
		//		},
		//		{
		//			name:         "R2",
		//			index:        1,
		//			isActive:     true,
		//			initialDelay: 0,
		//		},
		//		{
		//			name:         "R3",
		//			index:        2,
		//			isActive:     true,
		//			initialDelay: 0,
		//		},
		//		{
		//			name:         "R4",
		//			index:        3,
		//			isActive:     true,
		//			initialDelay: 0,
		//		},
		//	},
		//},
		{
			name: "basic test 2",
			testRelayers: []RelayerTestDescriber{
				{
					name:         "R1",
					index:        0,
					isActive:     true,
					initialDelay: 0,
				},
				{
					name:         "R2",
					index:        1,
					isActive:     true,
					initialDelay: 0,
				},
				{
					name:         "R3",
					index:        2,
					isActive:     true,
					initialDelay: 0,
				},
			},
		},
	}

	for _, t := range testCases {
		names := s.SetupIndividualTest(t)
		time.Sleep(3 * time.Second)

		cc := static.NewStaticCommunicationCoordinator(s.testHosts[0])
		coordinator, _ := cc.GetCoordinator(s.testSessionID)

		s.Run(t.name, func() {
			resultChan := make(chan peer.ID)

			for _, r := range t.testRelayers {
				rDescriber := r
				if rDescriber.isActive {
					go func() {
						if rDescriber.initialDelay > 0 {
							time.Sleep(rDescriber.initialDelay)
						}
						c, err := s.testBullyCoordinators[rDescriber.index].GetCoordinator(nil, names)
						s.Nil(err)
						resultChan <- c
					}()
				}
			}

			for i := 0; i < len(t.testRelayers); i++ {
				select {
				case c := <-resultChan:
					fmt.Printf("%s\n", names[c])
					s.Equal(coordinator, c)
				}
			}
		})
		// s.TearDownTest()
	}
}
