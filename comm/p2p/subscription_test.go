package p2p

import (
	comm "github.com/ChainSafe/chainbridge-core/comm"
	"github.com/stretchr/testify/suite"
	"testing"
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
	subscriptionID := subscriptionManager.subscribe("1", comm.CoordinatorPingMsg, sChannel)
	subscribers := subscriptionManager.getSubscribers("1", comm.CoordinatorPingMsg)
	s.Len(subscribers, 1)

	subscriptionManager.unSubscribe(subscriptionID)
	subscribers = subscriptionManager.getSubscribers("1", comm.CoordinatorPingMsg)
	s.Len(subscribers, 0)
}

func (s *SessionSubscriptionManagerTestSuite) TestSessionSubscriptionManager_ManageMultipleSubscribe_Success() {
	subscriptionManager := NewSessionSubscriptionManager()

	sub1Channel := make(chan *comm.WrappedMessage)
	subscriptionID1 := subscriptionManager.subscribe("1", comm.CoordinatorPingMsg, sub1Channel)

	sub2Channel := make(chan *comm.WrappedMessage)
	_ = subscriptionManager.subscribe("1", comm.CoordinatorPingMsg, sub2Channel)

	sub3Channel := make(chan *comm.WrappedMessage)
	subscriptionID3 := subscriptionManager.subscribe("2", comm.CoordinatorPingMsg, sub3Channel)

	subscribers := subscriptionManager.getSubscribers("1", comm.CoordinatorPingMsg)
	s.Len(subscribers, 2)

	subscribers = subscriptionManager.getSubscribers("2", comm.CoordinatorPingMsg)
	s.Len(subscribers, 1)

	subscriptionManager.unSubscribe(subscriptionID1)
	subscriptionManager.unSubscribe(subscriptionID3)

	subscribers = subscriptionManager.getSubscribers("1", comm.CoordinatorPingMsg)
	s.Len(subscribers, 1)

	subscribers = subscriptionManager.getSubscribers("2", comm.CoordinatorPingMsg)
	s.Len(subscribers, 0)
}
