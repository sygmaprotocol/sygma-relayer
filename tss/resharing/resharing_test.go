package resharing_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/comm/elector"
	"github.com/ChainSafe/sygma-relayer/keyshare"
	"github.com/ChainSafe/sygma-relayer/tss"
	"github.com/ChainSafe/sygma-relayer/tss/resharing"
	tsstest "github.com/ChainSafe/sygma-relayer/tss/test"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/suite"
)

type ResharingTestSuite struct {
	tsstest.CoordinatorTestSuite
}

func TestRunResharingTestSuite(t *testing.T) {
	suite.Run(t, new(ResharingTestSuite))
}

func (s *ResharingTestSuite) Test_ValidResharingProcess_OldAndNewSubset() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}

	hosts := []host.Host{}
	for i := 0; i < s.PartyNumber+1; i++ {
		host, _ := tsstest.NewHost(i)
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
		storer := keyshare.NewKeyshareStore(fmt.Sprintf("../test/keyshares/%d.keyshare", i))
		share, _ := storer.GetKeyshare()
		s.MockStorer.EXPECT().LockKeyshare()
		s.MockStorer.EXPECT().UnlockKeyshare()
		s.MockStorer.EXPECT().GetKeyshare().Return(share, nil)
		s.MockStorer.EXPECT().StoreKeyshare(gomock.Any()).Return(nil)
		resharing := resharing.NewResharing("resharing2", 1, host, &communication, s.MockStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.BullyConfig)
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, electorFactory))
		processes = append(processes, resharing)
	}
	tsstest.SetupCommunication(communicationMap)

	statusChn := make(chan error, s.PartyNumber)
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

func (s *ResharingTestSuite) Test_InvalidResharingProcess_InvalidOldThreshold_LessThenZero() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}

	hosts := []host.Host{}
	for i := 0; i < s.PartyNumber+1; i++ {
		host, _ := tsstest.NewHost(i)
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
		storer := keyshare.NewKeyshareStore(fmt.Sprintf("../test/keyshares/%d.keyshare", i))
		share, _ := storer.GetKeyshare()

		// set old threshold to invalid value
		share.Threshold = -1

		s.MockStorer.EXPECT().LockKeyshare()
		s.MockStorer.EXPECT().UnlockKeyshare()
		s.MockStorer.EXPECT().GetKeyshare().Return(share, nil)
		resharing := resharing.NewResharing("resharing3", 1, host, &communication, s.MockStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.BullyConfig)
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, electorFactory))
		processes = append(processes, resharing)
	}
	tsstest.SetupCommunication(communicationMap)

	statusChn := make(chan error, s.PartyNumber)
	resultChn := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	for i, coordinator := range coordinators {
		go coordinator.Execute(ctx, processes[i], resultChn, statusChn)
	}

	err := <-statusChn
	s.NotNil(err)
	s.Equal("process failed with error: threshold too small", err.Error())
	err = <-statusChn
	s.NotNil(err)
	s.Equal("process failed with error: threshold too small", err.Error())
	err = <-statusChn
	s.NotNil(err)
	s.Equal("process failed with error: threshold too small", err.Error())
	err = <-statusChn
	s.NotNil(err)
	s.Equal("process failed with error: threshold too small", err.Error())

	time.Sleep(time.Millisecond * 50)
	cancel()
}

func (s *ResharingTestSuite) Test_InvalidResharingProcess_InvalidOldThreshold_BiggerThenSubsetLength() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}

	hosts := []host.Host{}
	for i := 0; i < s.PartyNumber+1; i++ {
		host, _ := tsstest.NewHost(i)
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
		storer := keyshare.NewKeyshareStore(fmt.Sprintf("../test/keyshares/%d.keyshare", i))
		share, _ := storer.GetKeyshare()

		// set old threshold to invalid value
		share.Threshold = 314

		s.MockStorer.EXPECT().LockKeyshare()
		s.MockStorer.EXPECT().UnlockKeyshare()
		s.MockStorer.EXPECT().GetKeyshare().Return(share, nil)
		resharing := resharing.NewResharing("resharing4", 1, host, &communication, s.MockStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.BullyConfig)
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, electorFactory))
		processes = append(processes, resharing)
	}
	tsstest.SetupCommunication(communicationMap)

	statusChn := make(chan error, s.PartyNumber)
	resultChn := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	for i, coordinator := range coordinators {
		go coordinator.Execute(ctx, processes[i], resultChn, statusChn)
	}

	err := <-statusChn
	s.NotNil(err)
	s.Equal("process failed with error: threshold bigger then subset", err.Error())
	err = <-statusChn
	s.NotNil(err)
	s.Equal("process failed with error: threshold bigger then subset", err.Error())
	err = <-statusChn
	s.NotNil(err)
	s.Equal("process failed with error: threshold bigger then subset", err.Error())
	err = <-statusChn
	s.NotNil(err)
	s.Equal("process failed with error: threshold bigger then subset", err.Error())

	time.Sleep(time.Millisecond * 50)
	cancel()
}
