package p2p

import (
	"testing"

	comm "github.com/ChainSafe/sygma/comm"
	"github.com/stretchr/testify/suite"
)

type SessionSubscriptionManagerTestSuite struct {
	suite.Suite
}

func TestRunSessionSubscriptionManagerTestSuite(t *testing.T) {
	suite.Run(t, new(SessionSubscriptionManagerTestSuite))
}

func (s *SessionSubscriptionManagerTestSuite) SetupSuite()    {}
func (s *SessionSubscriptionManagerTestSuite) TearDownSuite() {}
func (s *SessionSubscriptionManagerTestSuite) SetupTest() {

}
func (s *SessionSubscriptionManagerTestSuite) TearDownTest() {}

func (s *SessionSubscriptionManagerTestSuite) TestSessionSubscriptionManager_ManageSingleSubscribe_Success() {
	subscriptionManager := NewSessionSubscriptionManager()

	sChannel := make(chan *comm.WrappedMessage)
	subscriptionID := subscriptionManager.Subscribe("1", comm.CoordinatorPingMsg, sChannel)
	subscribers := subscriptionManager.GetSubscribers("1", comm.CoordinatorPingMsg)
	s.Len(subscribers, 1)

	subscriptionManager.UnSubscribe(subscriptionID)
	subscribers = subscriptionManager.GetSubscribers("1", comm.CoordinatorPingMsg)
	s.Len(subscribers, 0)
}

func (s *SessionSubscriptionManagerTestSuite) TestSessionSubscriptionManager_ManageMultipleSubscribe_Success() {
	subscriptionManager := NewSessionSubscriptionManager()

	sub1Channel := make(chan *comm.WrappedMessage)
	subscriptionID1 := subscriptionManager.Subscribe("1", comm.CoordinatorPingMsg, sub1Channel)

	sub2Channel := make(chan *comm.WrappedMessage)
	_ = subscriptionManager.Subscribe("1", comm.CoordinatorPingMsg, sub2Channel)

	sub3Channel := make(chan *comm.WrappedMessage)
	subscriptionID3 := subscriptionManager.Subscribe("2", comm.CoordinatorPingMsg, sub3Channel)

	subscribers := subscriptionManager.GetSubscribers("1", comm.CoordinatorPingMsg)
	s.Len(subscribers, 2)

	subscribers = subscriptionManager.GetSubscribers("2", comm.CoordinatorPingMsg)
	s.Len(subscribers, 1)

	subscriptionManager.UnSubscribe(subscriptionID1)
	subscriptionManager.UnSubscribe(subscriptionID3)

	subscribers = subscriptionManager.GetSubscribers("1", comm.CoordinatorPingMsg)
	s.Len(subscribers, 1)

	subscribers = subscriptionManager.GetSubscribers("2", comm.CoordinatorPingMsg)
	s.Len(subscribers, 0)
}
