package tss_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/chainbridge-core/comm"
	mock_comm "github.com/ChainSafe/chainbridge-core/comm/mock"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/chainbridge-core/tss"
	"github.com/ChainSafe/chainbridge-core/tss/keygen"
	mock_keygen "github.com/ChainSafe/chainbridge-core/tss/keygen/mock"
	mock_tss "github.com/ChainSafe/chainbridge-core/tss/mock"
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
	gomockController  *gomock.Controller
	mockStorer        *mock_keygen.MockSaveDataStorer
	mockCommunication *mock_comm.MockCommunication
	mockTssProcess    *mock_tss.MockTssProcess

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
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, nil))
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
		keygen := keygen.NewKeygen("keygen", s.threshold, host, &communication, s.mockStorer)
		keygen.Timeout = time.Second * 5
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, nil))
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
		fetcher := store.NewKeyshareStore(fmt.Sprintf("./test/keyshares/%d.keyshare", i))

		msgBytes := []byte("Message")
		msg := big.NewInt(0)
		msg.SetBytes(msgBytes)
		signing, err := signing.NewSigning(msg, "signing", host, &communication, fetcher)
		if err != nil {
			panic(err)
		}
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, nil))
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
		fetcher := store.NewKeyshareStore(fmt.Sprintf("./test/keyshares/%d.keyshare", i))

		msgBytes := []byte("Message")
		msg := big.NewInt(0)
		msg.SetBytes(msgBytes)
		signing, err := signing.NewSigning(msg, "signing", host, &communication, fetcher)
		if err != nil {
			panic(err)
		}
		signing.Timeout = time.Millisecond * 200
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, nil))
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
	err = <-statusChn
	s.NotNil(err)
	err = <-statusChn
	s.NotNil(err)
	cancel()
}

/*
func (s *CoordinatorTestSuite) Test_CoordinatorOffline_RetryProcessWithBully() {
	s.mockTssProcess.EXPECT().SessionID().Return("sessionID").AnyTimes()
	s.mockTssProcess.EXPECT().Stop().Return().AnyTimes()
	s.mockCommunication.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).Return(comm.NewSubscriptionID("sessionID", comm.TssReadyMsg)).AnyTimes()
	s.mockCommunication.EXPECT().UnSubscribe(gomock.Any()).Return().AnyTimes()
	s.mockCommunication.EXPECT().Broadcast(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), nil).Return().AnyTimes()
	nil.EXPECT().Coordinator(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(excludedPeers peer.IDSlice, coordinatorChan chan peer.ID, errChan chan error) {
		go func() {
			for {
				coordinatorChan <- peer.ID("QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX")
				break
			}
		}()
	}).Times(2)

	coordinators := []*tss.Coordinator{}
	for _, host := range s.hosts {
		coordinator := tss.NewCoordinator(host, s.mockCommunication, nil)
		coordinator.CoordinatorTimeout = time.Millisecond * 30
		coordinators = append(coordinators, coordinator)
	}

	statusChn := make(chan error, s.partyNumber)
	resultChn := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	for _, coordinator := range coordinators {
		go coordinator.Execute(ctx, s.mockTssProcess, resultChn, statusChn)
	}

	err := <-statusChn
	s.NotNil(err)
	err = <-statusChn
	s.NotNil(err)
	cancel()
}

func (s *CoordinatorTestSuite) Test_TssError_RetryProcessWithBully() {
	s.mockTssProcess.EXPECT().SessionID().Return("sessionID").AnyTimes()
	s.mockTssProcess.EXPECT().Stop().Return().AnyTimes()
	s.mockTssProcess.EXPECT().Ready(gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
	s.mockTssProcess.EXPECT().StartParams(gomock.Any()).Return([]string{}).AnyTimes()

	nil.EXPECT().Coordinator(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(excludedPeers peer.IDSlice, coordinatorChan chan peer.ID, errChan chan error) {
		go func() {
			for {
				coordinatorChan <- s.hosts[0].ID()
				break
			}
		}()
	}).AnyTimes()
	s.mockTssProcess.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Do(func(ctx context.Context, resultChn chan interface{}, errChn chan error, params []string) {
		go func() {
			for {
				errChn <- tssLib.NewError(errors.New("error"), "signing", 1, common.CreatePartyID("QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX"), common.CreatePartyID("QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX"))
				break
			}
		}()
	}).AnyTimes()

	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	for _, host := range s.hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, nil))
	}
	setupCommunication(communicationMap)

	statusChn := make(chan error, s.partyNumber)
	resultChn := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	for _, coordinator := range coordinators {
		go coordinator.Execute(ctx, s.mockTssProcess, resultChn, statusChn)
	}

	err := <-statusChn
	s.NotNil(err)
	err = <-statusChn
	s.NotNil(err)
	err = <-statusChn
	s.NotNil(err)
	cancel()
}
*/

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
		keygen := keygen.NewKeygen("keygen", s.threshold, host, &communication, s.mockStorer)
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, nil))
		processes = append(processes, keygen)
	}
	setupCommunication(communicationMap)

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
	cancel()
}
