package p2p

import (
	"sync"

	comm "github.com/ChainSafe/sygma/comm"
)

// SessionSubscriptionManager manages channel subscriptions by comm.SessionID
type SessionSubscriptionManager struct {
	lock *sync.Mutex
	// sessionID -> messageType -> subscriptionID
	subscribersMap map[string]map[comm.ChainBridgeMessageType]map[string]chan *comm.WrappedMessage
}

func NewSessionSubscriptionManager() SessionSubscriptionManager {
	return SessionSubscriptionManager{
		lock: &sync.Mutex{},
		subscribersMap: make(
			map[string]map[comm.ChainBridgeMessageType]map[string]chan *comm.WrappedMessage,
		),
	}
}

func (ms *SessionSubscriptionManager) getSubscribers(
	sessionID string,
	msgType comm.ChainBridgeMessageType,
) []chan *comm.WrappedMessage {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	subsAsMap, ok := ms.subscribersMap[sessionID][msgType]
	if !ok {
		return []chan *comm.WrappedMessage{}
	}
	var subsAsArray []chan *comm.WrappedMessage
	for _, sub := range subsAsMap {
		subsAsArray = append(subsAsArray, sub)
	}
	return subsAsArray
}

func (ms *SessionSubscriptionManager) subscribe(
	sessionID string, msgType comm.ChainBridgeMessageType, channel chan *comm.WrappedMessage,
) comm.SubscriptionID {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	_, ok := ms.subscribersMap[sessionID]
	if !ok {
		ms.subscribersMap[sessionID] =
			map[comm.ChainBridgeMessageType]map[string]chan *comm.WrappedMessage{}
	}

	_, ok = ms.subscribersMap[sessionID][msgType]
	if !ok {
		ms.subscribersMap[sessionID][msgType] =
			map[string]chan *comm.WrappedMessage{}
	}

	subID := comm.NewSubscriptionID(sessionID, msgType)
	ms.subscribersMap[sessionID][msgType][subID.SubscriptionIdentifier()] = channel
	return subID
}

func (ms *SessionSubscriptionManager) unSubscribe(
	subscriptionID comm.SubscriptionID,
) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	sessionID, msgType, subID, err := subscriptionID.Unwrap()
	if err != nil {
		return
	}

	_, ok := ms.subscribersMap[sessionID][msgType][subID]
	if !ok {
		return
	}

	delete(ms.subscribersMap[sessionID][msgType], subID)
}
