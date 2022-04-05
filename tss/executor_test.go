package tss_test

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/ChainSafe/chainbridge-core/tss"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/ChainSafe/chainbridge-core/tss/keygen"
	mock_keygen "github.com/ChainSafe/chainbridge-core/tss/keygen/mock"
	tsstest "github.com/ChainSafe/chainbridge-core/tss/test"
	"github.com/golang/mock/gomock"
	libp2p "github.com/libp2p/go-libp2p"
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

type ExecutorTestSutite struct {
	suite.Suite
	gomockController *gomock.Controller
	mockStorer       *mock_keygen.MockSaveDataStorer

	hosts       []host.Host
	threshold   int
	partyNumber int
}

func TestRunExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorTestSutite))
}

func (s *ExecutorTestSutite) SetupTest() {
	s.gomockController = gomock.NewController(s.T())
	s.mockStorer = mock_keygen.NewMockSaveDataStorer(s.gomockController)

	s.partyNumber = 3
	s.threshold = 1

	for i := 0; i < s.partyNumber; i++ {
		host, _ := NewHost()
		s.hosts = append(s.hosts, host)
	}
	for _, host := range s.hosts {
		for _, peer := range s.hosts {
			host.Peerstore().AddAddr(peer.ID(), peer.Addrs()[0], peerstore.PermanentAddrTTL)
		}
	}
}

func (s *ExecutorTestSutite) Test_ValidKeygenProcess() {
	errChn := make(chan error)
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	for _, host := range s.hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[string]chan *common.WrappedMessage),
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
