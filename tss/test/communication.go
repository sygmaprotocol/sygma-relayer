package tsstest

import (
	"fmt"
	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

type Receiver interface {
	ReceiveMessage(msg *communication.WrappedMessage, topic communication.ChainBridgeMessageType, sessionID string)
}

type TestCommunication struct {
	Host               host.Host
	Subscriptions      map[communication.SubscriptionID]chan *communication.WrappedMessage
	PeerCommunications map[string]Receiver
}

func (tc *TestCommunication) Broadcast(
	peers peer.IDSlice,
	msg []byte,
	msgType communication.ChainBridgeMessageType,
	sessionID string,
	errChan chan error,
) {
	wMsg := communication.WrappedMessage{
		MessageType: msgType,
		SessionID:   sessionID,
		Payload:     msg,
		From:        tc.Host.ID(),
	}
	for _, peer := range peers {
		if tc.PeerCommunications[peer.Pretty()] == nil {
			continue
		}

		tc.PeerCommunications[peer.Pretty()].ReceiveMessage(&wMsg, msgType, sessionID)
	}
}

func (ts *TestCommunication) Subscribe(
	sessionID string,
	topic communication.ChainBridgeMessageType,
	channel chan *communication.WrappedMessage,
) communication.SubscriptionID {
	subID := communication.SubscriptionID(fmt.Sprintf("%s-%s", sessionID, topic))
	ts.Subscriptions[subID] = channel
	return subID
}

func (ts *TestCommunication) UnSubscribe(subscriptionID communication.SubscriptionID) {
	delete(ts.Subscriptions, subscriptionID)
}

func (ts *TestCommunication) ReceiveMessage(msg *communication.WrappedMessage, topic communication.ChainBridgeMessageType, sessionID string) {
	ts.Subscriptions[communication.SubscriptionID(fmt.Sprintf("%s-%s", sessionID, topic))] <- msg
}
