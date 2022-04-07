package p2p

import (
	comm "github.com/ChainSafe/chainbridge-core/communication"
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
	subscriptionID := subscriptionManager.Subscribe("1", sChannel)
	subscribers := subscriptionManager.GetSubscribers("1")
	s.Len(subscribers, 1)

	subscriptionManager.UnSubscribe("1", subscriptionID)
	subscribers = subscriptionManager.GetSubscribers("1")
	s.Len(subscribers, 0)
}

func (s *SessionSubscriptionManagerTestSuite) TestSessionSubscriptionManager_ManageMultipleSubscribe_Success() {
	subscriptionManager := NewSessionSubscriptionManager()

	sub1Channel := make(chan *comm.WrappedMessage)
	subscriptionID1 := subscriptionManager.Subscribe("1", sub1Channel)

	sub2Channel := make(chan *comm.WrappedMessage)
	_ = subscriptionManager.Subscribe("1", sub2Channel)

	sub3Channel := make(chan *comm.WrappedMessage)
	subscriptionID3 := subscriptionManager.Subscribe("2", sub3Channel)

	subscribers := subscriptionManager.GetSubscribers("1")
	s.Len(subscribers, 2)

	subscribers = subscriptionManager.GetSubscribers("2")
	s.Len(subscribers, 1)

	subscriptionManager.UnSubscribe("1", subscriptionID1)
	subscriptionManager.UnSubscribe("2", subscriptionID3)

	subscribers = subscriptionManager.GetSubscribers("1")
	s.Len(subscribers, 1)

	subscribers = subscriptionManager.GetSubscribers("2")
	s.Len(subscribers, 0)
}
