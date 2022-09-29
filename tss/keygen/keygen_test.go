package keygen_test

import (
	"context"
	"testing"
	"time"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/comm/elector"
	"github.com/ChainSafe/sygma-relayer/tss"
	"github.com/ChainSafe/sygma-relayer/tss/keygen"
	tsstest "github.com/ChainSafe/sygma-relayer/tss/test"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/suite"
)

type KeygenTestSuite struct {
	tsstest.CoordinatorTestSuite
}

func TestRunKeygenTestSuite(t *testing.T) {
	suite.Run(t, new(KeygenTestSuite))
}

func (s *KeygenTestSuite) Test_ValidKeygenProcess() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}

	for _, host := range s.CoordinatorTestSuite.Hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		keygen := keygen.NewKeygen("keygen", s.Threshold, host, &communication, s.MockStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.BullyConfig)
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, electorFactory))
		processes = append(processes, keygen)
	}
	tsstest.SetupCommunication(communicationMap)

	s.MockStorer.EXPECT().LockKeyshare().Times(3)
	s.MockStorer.EXPECT().UnlockKeyshare().Times(3)
	s.MockStorer.EXPECT().StoreKeyshare(gomock.Any()).Times(3)
	status := make(chan error, s.PartyNumber)
	ctx, cancel := context.WithCancel(context.Background())
	for i, coordinator := range coordinators {
		go coordinator.Execute(ctx, processes[i], nil, status)
	}

	for i := 0; i < s.PartyNumber; i++ {
		err := <-status
		s.Nil(err)
	}
	time.Sleep(time.Millisecond * 50)
	cancel()
}

func (s *KeygenTestSuite) Test_KeygenTimeout() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}
	for _, host := range s.CoordinatorTestSuite.Hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		keygen := keygen.NewKeygen("keygen2", s.Threshold, host, &communication, s.MockStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.BullyConfig)
		coordinator := tss.NewCoordinator(host, &communication, electorFactory)
		coordinator.TssTimeout = time.Millisecond
		coordinators = append(coordinators, coordinator)
		processes = append(processes, keygen)
	}
	tsstest.SetupCommunication(communicationMap)

	s.MockStorer.EXPECT().LockKeyshare().AnyTimes()
	s.MockStorer.EXPECT().UnlockKeyshare().AnyTimes()
	s.MockStorer.EXPECT().StoreKeyshare(gomock.Any()).Times(0)
	status := make(chan error, s.PartyNumber)
	ctx, cancel := context.WithCancel(context.Background())
	for i, coordinator := range coordinators {
		go coordinator.Execute(ctx, processes[i], nil, status)
	}

	for i := 0; i < s.PartyNumber; i++ {
		err := <-status
		s.NotNil(err)
	}
	time.Sleep(time.Millisecond * 50)
	cancel()
}
