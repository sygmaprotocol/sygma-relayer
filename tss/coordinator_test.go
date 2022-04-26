package tss_test

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/ChainSafe/chainbridge-core/tss"
	"github.com/ChainSafe/chainbridge-core/tss/keygen"
	mock_keygen "github.com/ChainSafe/chainbridge-core/tss/keygen/mock"
	tsstest "github.com/ChainSafe/chainbridge-core/tss/test"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/stretchr/testify/suite"
)

func NewHost() (host.Host, error) {
	priv, _, err := crypto.GenerateRSAKeyPair(2048, rand.Reader)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}

	return h, nil
}

type CoordinatorTestSuite struct {
	suite.Suite
	gomockController *gomock.Controller
	mockStorer       *mock_keygen.MockSaveDataStorer

	threshold   int
	partyNumber int
}

func TestRunCoordinatorTestSuite(t *testing.T) {
	suite.Run(t, new(CoordinatorTestSuite))
}

func (s *CoordinatorTestSuite) SetupTest() {
	s.gomockController = gomock.NewController(s.T())
	s.mockStorer = mock_keygen.NewMockSaveDataStorer(s.gomockController)

	s.partyNumber = 3
	s.threshold = 1
}

func (s *CoordinatorTestSuite) Test_ValidKeygenProcess() {
	errChn := make(chan error)
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}

	hosts := []host.Host{}
	for i := 0; i < s.partyNumber; i++ {
		host, _ := NewHost()
		hosts = append(hosts, host)
	}
	for _, host := range hosts {
		for _, peer := range hosts {
			host.Peerstore().AddAddr(peer.ID(), peer.Addrs()[0], peerstore.PermanentAddrTTL)
		}
	}
	for _, host := range hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[communication.SubscriptionID]chan *communication.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		keygen := keygen.NewKeygen("keygen", s.threshold, host, &communication, s.mockStorer, errChn)
		coordinators = append(coordinators, tss.NewCoordinator(host, keygen, &communication, errChn))
	}
	for self, comm := range communicationMap {
		peerComms := make(map[string]tsstest.Receiver)
		for p, otherComm := range communicationMap {
			if self.Pretty() == p.Pretty() {
				continue
			}
			peerComms[p.Pretty()] = otherComm
		}
		comm.PeerCommunications = peerComms
	}

	s.mockStorer.EXPECT().LockKeyshare().Times(3)
	s.mockStorer.EXPECT().UnlockKeyshare().Times(3)
	s.mockStorer.EXPECT().StoreKeyshare(gomock.Any()).Times(3)
	status := make(chan error, s.partyNumber)
	ctx, cancel := context.WithCancel(context.Background())
	for _, coordinator := range coordinators {
		go coordinator.Execute(ctx, status)
	}

	for i := 0; i < s.partyNumber; i++ {
		err := <-status
		s.Nil(err)
	}
	cancel()
}

func (s *CoordinatorTestSuite) Test_KeygenTimeoutOut() {
	errChn := make(chan error)
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	hosts := []host.Host{}
	for i := 0; i < s.partyNumber; i++ {
		host, _ := NewHost()
		hosts = append(hosts, host)
	}
	for _, host := range hosts {
		for _, peer := range hosts {
			host.Peerstore().AddAddr(peer.ID(), peer.Addrs()[0], peerstore.PermanentAddrTTL)
		}
	}
	for _, host := range hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[communication.SubscriptionID]chan *communication.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		keygen := keygen.NewKeygen("keygen", s.threshold, host, &communication, s.mockStorer, errChn)
		keygen.Timeout = time.Second * 5
		coordinators = append(coordinators, tss.NewCoordinator(host, keygen, &communication, errChn))
	}
	for self, comm := range communicationMap {
		peerComms := make(map[string]tsstest.Receiver)
		for p, otherComm := range communicationMap {
			if self.Pretty() == p.Pretty() {
				continue
			}
			peerComms[p.Pretty()] = otherComm
		}
		comm.PeerCommunications = peerComms
	}

	s.mockStorer.EXPECT().LockKeyshare().Times(3)
	s.mockStorer.EXPECT().UnlockKeyshare().Times(3)
	s.mockStorer.EXPECT().StoreKeyshare(gomock.Any()).Times(0)
	status := make(chan error, s.partyNumber)
	ctx, cancel := context.WithCancel(context.Background())
	for _, coordinator := range coordinators {
		go coordinator.Execute(ctx, status)
	}

	for i := 0; i < s.partyNumber; i++ {
		err := <-status
		s.NotNil(err)
	}
	cancel()
}
