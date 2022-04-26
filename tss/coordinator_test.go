package tss_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/chainbridge-core/tss"
	"github.com/ChainSafe/chainbridge-core/tss/keygen"
	mock_keygen "github.com/ChainSafe/chainbridge-core/tss/keygen/mock"
	"github.com/ChainSafe/chainbridge-core/tss/signing"
	tsstest "github.com/ChainSafe/chainbridge-core/tss/test"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/stretchr/testify/suite"
)

func newHost(i int) (host.Host, error) {
	privBytes, err := ioutil.ReadFile(fmt.Sprintf("./test/pks/%d.pk", i))
	if err != nil {
		return nil, err
	}

	priv, err := crypto.UnmarshalPrivateKey(privBytes)
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

func setupCommunication(commMap map[peer.ID]*tsstest.TestCommunication) {
	for self, comm := range commMap {
		peerComms := make(map[string]tsstest.Receiver)
		for p, otherComm := range commMap {
			if self.Pretty() == p.Pretty() {
				continue
			}
			peerComms[p.Pretty()] = otherComm
		}
		comm.PeerCommunications = peerComms
	}
}

type CoordinatorTestSuite struct {
	suite.Suite
	gomockController *gomock.Controller
	mockStorer       *mock_keygen.MockSaveDataStorer

	hosts       []host.Host
	threshold   int
	partyNumber int
}

func TestRunCoordinatorTestSuite(t *testing.T) {
	suite.Run(t, new(CoordinatorTestSuite))
}

func (s *CoordinatorTestSuite) SetupSuite() {
	s.gomockController = gomock.NewController(s.T())
	s.mockStorer = mock_keygen.NewMockSaveDataStorer(s.gomockController)

	s.partyNumber = 3
	s.threshold = 1

	hosts := []host.Host{}
	for i := 0; i < s.partyNumber; i++ {
		host, _ := newHost(i)
		hosts = append(hosts, host)
	}
	for _, host := range hosts {
		for _, peer := range hosts {
			host.Peerstore().AddAddr(peer.ID(), peer.Addrs()[0], peerstore.PermanentAddrTTL)
		}
	}
	s.hosts = hosts
}

func (s *CoordinatorTestSuite) Test_ValidKeygenProcess() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}

	for _, host := range s.hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[communication.SubscriptionID]chan *communication.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		keygen := keygen.NewKeygen("keygen", s.threshold, host, &communication, s.mockStorer)
		coordinators = append(coordinators, tss.NewCoordinator(host, keygen, &communication))
	}
	setupCommunication(communicationMap)

	s.mockStorer.EXPECT().LockKeyshare().Times(3)
	s.mockStorer.EXPECT().UnlockKeyshare().Times(3)
	s.mockStorer.EXPECT().StoreKeyshare(gomock.Any()).Times(3)
	status := make(chan error, s.partyNumber)
	ctx, cancel := context.WithCancel(context.Background())
	for _, coordinator := range coordinators {
		go coordinator.Execute(ctx, nil, status)
	}

	for i := 0; i < s.partyNumber; i++ {
		err := <-status
		s.Nil(err)
	}
	cancel()
}

func (s *CoordinatorTestSuite) Test_KeygenTimeout() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	for _, host := range s.hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[communication.SubscriptionID]chan *communication.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		keygen := keygen.NewKeygen("keygen", s.threshold, host, &communication, s.mockStorer)
		keygen.Timeout = time.Second * 5
		coordinators = append(coordinators, tss.NewCoordinator(host, keygen, &communication))
	}
	setupCommunication(communicationMap)

	s.mockStorer.EXPECT().LockKeyshare().Times(3)
	s.mockStorer.EXPECT().UnlockKeyshare().Times(3)
	s.mockStorer.EXPECT().StoreKeyshare(gomock.Any()).Times(0)
	status := make(chan error, s.partyNumber)
	ctx, cancel := context.WithCancel(context.Background())
	for _, coordinator := range coordinators {
		go coordinator.Execute(ctx, nil, status)
	}

	for i := 0; i < s.partyNumber; i++ {
		err := <-status
		s.NotNil(err)
	}
	cancel()
}

func (s *CoordinatorTestSuite) Test_ValidSigningProcess() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}

	for i, host := range s.hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[string]chan *communication.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		fetcher := store.NewKeyshareStore(fmt.Sprintf("./test/keyshares/%d.keyshare", i))

		msgBytes := []byte("Message")
		msg := big.NewInt(0)
		msg.SetBytes(msgBytes)
		signing, err := signing.NewSigning(msg, "signing", host, &communication, fetcher)
		if err != nil {
			panic(err)
		}
		coordinators = append(coordinators, tss.NewCoordinator(host, signing, &communication))
	}
	setupCommunication(communicationMap)

	statusChn := make(chan error, s.partyNumber)
	resultChn := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	for _, coordinator := range coordinators {
		go coordinator.Execute(ctx, resultChn, statusChn)
	}

	err := <-statusChn
	s.Nil(err)
	sig1 := <-resultChn
	sig2 := <-resultChn
	s.Equal(sig1, sig2)
	cancel()
}

func (s *CoordinatorTestSuite) Test_SigningTimeout() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}

	for i, host := range s.hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[string]chan *communication.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		fetcher := store.NewKeyshareStore(fmt.Sprintf("./test/keyshares/%d.keyshare", i))

		msgBytes := []byte("Message")
		msg := big.NewInt(0)
		msg.SetBytes(msgBytes)
		signing, err := signing.NewSigning(msg, "signing", host, &communication, fetcher)
		if err != nil {
			panic(err)
		}
		signing.Timeout = time.Millisecond * 200
		coordinators = append(coordinators, tss.NewCoordinator(host, signing, &communication))
	}
	setupCommunication(communicationMap)

	statusChn := make(chan error, s.partyNumber)
	resultChn := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	for _, coordinator := range coordinators {
		go coordinator.Execute(ctx, resultChn, statusChn)
	}

	err := <-statusChn
	s.Nil(err)
	err = <-statusChn
	s.NotNil(err)
	err = <-statusChn
	s.NotNil(err)
	cancel()
}
