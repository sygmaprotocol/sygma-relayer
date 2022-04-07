package p2p

import (
	comm "github.com/ChainSafe/chainbridge-core/communication"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

// SessionSubscriptionManager manages channel subscriptions by comm.SessionID
type SessionSubscriptionManager struct {
	lock        *sync.Mutex
	r           rand.Source
	subscribers map[comm.SessionID]map[comm.SubscriptionID]chan *comm.WrappedMessage
}

func NewSessionSubscriptionManager() *SessionSubscriptionManager {
	return &SessionSubscriptionManager{
		lock:        &sync.Mutex{},
		r:           rand.NewSource(time.Now().Unix()),
		subscribers: make(map[comm.SessionID]map[comm.SubscriptionID]chan *comm.WrappedMessage),
	}
}

// GetSubscribers return all subscriptionManagers for specific session
func (ms *SessionSubscriptionManager) GetSubscribers(
	sessionID comm.SessionID,
) []chan *comm.WrappedMessage {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	subsAsMap, ok := ms.subscribers[sessionID]
	if !ok {
		return nil
	}
	var subsAsArray []chan *comm.WrappedMessage
	for _, sub := range subsAsMap {
		subsAsArray = append(subsAsArray, sub)
	}
	return subsAsArray
}

// Subscribe adds provided channel to session subscriptionManagers.
// Returns SubscriptionID that is unique identifier of this subscription and is needed to UnSubscribe.
func (ms *SessionSubscriptionManager) Subscribe(
	sessionID comm.SessionID, channel chan *comm.WrappedMessage,
) comm.SubscriptionID {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	// create subscription id
	subID := comm.SubscriptionID(strconv.FormatInt(ms.r.Int63(), 10))
	sessionSubscribers, ok := ms.subscribers[sessionID]
	if !ok {
		sessionSubscribers = map[comm.SubscriptionID]chan *comm.WrappedMessage{}
	}
	sessionSubscribers[subID] = channel
	ms.subscribers[sessionID] = sessionSubscribers
	return subID
}

// UnSubscribe a specific subscription, defined by SubscriptionID
func (ms *SessionSubscriptionManager) UnSubscribe(
	sessionID comm.SessionID, subscriptionID comm.SubscriptionID,
) {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	_, ok := ms.subscribers[sessionID]
	if !ok {
		return
	}
	delete(ms.subscribers[sessionID], subscriptionID)
}
