package tsstest

import (
	"fmt"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

type Receiver interface {
	ReceiveMessage(msg *comm.WrappedMessage, topic comm.MessageType, sessionID string)
}

type TestCommunication struct {
	Host               host.Host
	Subscriptions      map[comm.SubscriptionID]chan *comm.WrappedMessage
	PeerCommunications map[string]Receiver
}

func (tc *TestCommunication) Broadcast(
	peers peer.IDSlice,
	msg []byte,
	msgType comm.MessageType,
	sessionID string,
	errChan chan error,
) {
	wMsg := comm.WrappedMessage{
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
	topic comm.MessageType,
	channel chan *comm.WrappedMessage,
) comm.SubscriptionID {
	subID := comm.SubscriptionID(fmt.Sprintf("%s-%s", sessionID, topic))
	ts.Subscriptions[subID] = channel
	return subID
}

func (ts *TestCommunication) UnSubscribe(subscriptionID comm.SubscriptionID) {
	delete(ts.Subscriptions, subscriptionID)
}

func (ts *TestCommunication) ReceiveMessage(msg *comm.WrappedMessage, topic comm.MessageType, sessionID string) {
	ts.Subscriptions[comm.SubscriptionID(fmt.Sprintf("%s-%s", sessionID, topic))] <- msg
}
