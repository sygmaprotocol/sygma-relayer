package tss_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/sygma/config/relayer"
	"github.com/ChainSafe/sygma/keyshare"

	"github.com/ChainSafe/sygma/comm"
	"github.com/ChainSafe/sygma/comm/elector"
	mock_comm "github.com/ChainSafe/sygma/comm/mock"
	"github.com/ChainSafe/sygma/tss"
	"github.com/ChainSafe/sygma/tss/keygen"
	mock_tss "github.com/ChainSafe/sygma/tss/mock"
	"github.com/ChainSafe/sygma/tss/resharing"
	"github.com/ChainSafe/sygma/tss/signing"
	tsstest "github.com/ChainSafe/sygma/tss/test"
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
	gomockController  *gomock.Controller
	mockStorer        *mock_tss.MockSaveDataStorer
	mockCommunication *mock_comm.MockCommunication
	mockTssProcess    *mock_tss.MockTssProcess

	hosts       []host.Host
	threshold   int
	partyNumber int
	bullyConfig relayer.BullyConfig
}

func TestRunCoordinatorTestSuite(t *testing.T) {
	suite.Run(t, new(CoordinatorTestSuite))
}

func (s *CoordinatorTestSuite) SetupTest() {
	s.gomockController = gomock.NewController(s.T())
	s.mockStorer = mock_tss.NewMockSaveDataStorer(s.gomockController)
	s.mockCommunication = mock_comm.NewMockCommunication(s.gomockController)
	s.mockTssProcess = mock_tss.NewMockTssProcess(s.gomockController)
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
	s.bullyConfig = relayer.BullyConfig{
		PingWaitTime:     1 * time.Second,
		PingBackOff:      1 * time.Second,
		PingInterval:     1 * time.Second,
		ElectionWaitTime: 2 * time.Second,
		BullyWaitTime:    25 * time.Second,
	}
}

func (s *CoordinatorTestSuite) Test_ValidKeygenProcess() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}

	for _, host := range s.hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		keygen := keygen.NewKeygen("keygen", s.threshold, host, &communication, s.mockStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.bullyConfig)
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, electorFactory))
		processes = append(processes, keygen)
	}
	setupCommunication(communicationMap)

	s.mockStorer.EXPECT().LockKeyshare().Times(3)
	s.mockStorer.EXPECT().UnlockKeyshare().Times(3)
	s.mockStorer.EXPECT().StoreKeyshare(gomock.Any()).Times(3)
	status := make(chan error, s.partyNumber)
	ctx, cancel := context.WithCancel(context.Background())
	for i, coordinator := range coordinators {
		go coordinator.Execute(ctx, processes[i], nil, status)
	}

	for i := 0; i < s.partyNumber; i++ {
		err := <-status
		s.Nil(err)
	}
	time.Sleep(time.Millisecond * 50)
	cancel()
}

func (s *CoordinatorTestSuite) Test_KeygenTimeout() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}
	for _, host := range s.hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		keygen := keygen.NewKeygen("keygen2", s.threshold, host, &communication, s.mockStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.bullyConfig)
		coordinator := tss.NewCoordinator(host, &communication, electorFactory)
		coordinator.TssTimeout = time.Millisecond
		coordinators = append(coordinators, coordinator)
		processes = append(processes, keygen)
	}
	setupCommunication(communicationMap)

	s.mockStorer.EXPECT().LockKeyshare().Times(3)
	s.mockStorer.EXPECT().UnlockKeyshare().Times(3)
	s.mockStorer.EXPECT().StoreKeyshare(gomock.Any()).Times(0)
	status := make(chan error, s.partyNumber)
	ctx, cancel := context.WithCancel(context.Background())
	for i, coordinator := range coordinators {
		go coordinator.Execute(ctx, processes[i], nil, status)
	}

	for i := 0; i < s.partyNumber; i++ {
		err := <-status
		s.NotNil(err)
	}
	time.Sleep(time.Millisecond * 50)
	cancel()
}

func (s *CoordinatorTestSuite) Test_ValidSigningProcess() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}

	for i, host := range s.hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		fetcher := keyshare.NewKeyshareStore(fmt.Sprintf("./test/keyshares/%d.keyshare", i))

		msgBytes := []byte("Message")
		msg := big.NewInt(0)
		msg.SetBytes(msgBytes)
		signing, err := signing.NewSigning(msg, "signing1", host, &communication, fetcher)
		if err != nil {
			panic(err)
		}
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.bullyConfig)
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, electorFactory))
		processes = append(processes, signing)
	}
	setupCommunication(communicationMap)

	statusChn := make(chan error, s.partyNumber)
	resultChn := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	for i, coordinator := range coordinators {
		go coordinator.Execute(ctx, processes[i], resultChn, statusChn)
	}

	err := <-statusChn
	s.Nil(err)
	sig := <-resultChn
	s.NotNil(sig)
	time.Sleep(time.Millisecond * 50)
	cancel()
}

func (s *CoordinatorTestSuite) Test_SigningTimeout() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}

	for i, host := range s.hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		fetcher := keyshare.NewKeyshareStore(fmt.Sprintf("./test/keyshares/%d.keyshare", i))

		msgBytes := []byte("Message")
		msg := big.NewInt(0)
		msg.SetBytes(msgBytes)
		signing, err := signing.NewSigning(msg, "signing2", host, &communication, fetcher)
		if err != nil {
			panic(err)
		}
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.bullyConfig)
		coordinator := tss.NewCoordinator(host, &communication, electorFactory)
		coordinator.TssTimeout = time.Nanosecond
		coordinators = append(coordinators, coordinator)
		processes = append(processes, signing)
	}
	setupCommunication(communicationMap)

	statusChn := make(chan error, s.partyNumber)
	resultChn := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	for i, coordinator := range coordinators {
		go coordinator.Execute(ctx, processes[i], resultChn, statusChn)
	}

	err := <-statusChn
	s.NotNil(err)
	err = <-statusChn
	s.NotNil(err)
	err = <-statusChn
	s.NotNil(err)
	time.Sleep(time.Millisecond * 50)
	cancel()
}

func (s *CoordinatorTestSuite) Test_PendingProcessExists() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}
	for _, host := range s.hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		keygen := keygen.NewKeygen("keygen3", s.threshold, host, &communication, s.mockStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.bullyConfig)
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, electorFactory))
		processes = append(processes, keygen)
	}
	setupCommunication(communicationMap)

	s.mockStorer.EXPECT().LockKeyshare().AnyTimes()
	status := make(chan error, s.partyNumber)
	ctx, cancel := context.WithCancel(context.Background())
	for i, coordinator := range coordinators {
		go coordinator.Execute(ctx, processes[i], nil, nil)
		time.Sleep(time.Millisecond * 50)
		go coordinator.Execute(ctx, processes[i], nil, status)
	}

	for i := 0; i < s.partyNumber; i++ {
		err := <-status
		s.Nil(err)
	}
	time.Sleep(time.Millisecond * 50)
	cancel()
}

func (s *CoordinatorTestSuite) Test_ValidResharingProcess_OldAndNewSubset() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}

	hosts := []host.Host{}
	for i := 0; i < s.partyNumber+1; i++ {
		host, _ := newHost(i)
		hosts = append(hosts, host)
	}
	for _, host := range hosts {
		for _, peer := range hosts {
			host.Peerstore().AddAddr(peer.ID(), peer.Addrs()[0], peerstore.PermanentAddrTTL)
		}
	}

	for i, host := range hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		storer := keyshare.NewKeyshareStore(fmt.Sprintf("./test/keyshares/%d.keyshare", i))
		share, _ := storer.GetKeyshare()
		s.mockStorer.EXPECT().LockKeyshare()
		s.mockStorer.EXPECT().UnlockKeyshare()
		s.mockStorer.EXPECT().GetKeyshare().Return(share, nil)
		s.mockStorer.EXPECT().StoreKeyshare(gomock.Any()).Return(nil)
		resharing := resharing.NewResharing("resharing2", 1, host, &communication, s.mockStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.bullyConfig)
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, electorFactory))
		processes = append(processes, resharing)
	}
	setupCommunication(communicationMap)

	statusChn := make(chan error, s.partyNumber)
	resultChn := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	for i, coordinator := range coordinators {
		go coordinator.Execute(ctx, processes[i], resultChn, statusChn)
	}

	err := <-statusChn
	s.Nil(err)
	err = <-statusChn
	s.Nil(err)
	err = <-statusChn
	s.Nil(err)
	err = <-statusChn
	s.Nil(err)

	time.Sleep(time.Millisecond * 50)
	cancel()
}
