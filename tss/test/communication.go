package tsstest

import (
	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

type Receiver interface {
	ReceiveMessage(msg *communication.WrappedMessage, topic communication.ChainBridgeMessageType, sessionID string)
}

type TestCommunication struct {
	Host               host.Host
	Subscriptions      map[string]chan *communication.WrappedMessage
	PeerCommunications map[string]Receiver
}

func (tc *TestCommunication) Broadcast(
	peers peer.IDSlice,
	msg []byte,
	msgType communication.ChainBridgeMessageType,
	sessionID string,
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
	topic communication.ChainBridgeMessageType,
	sessionID string,
	channel chan *communication.WrappedMessage,
) string {
	ts.Subscriptions[string(topic)+sessionID] = channel
	return string(topic) + sessionID
}

func (ts *TestCommunication) EndSession(sessionID string) {
	ts.Subscriptions = make(map[string]chan *communication.WrappedMessage)
}

func (ts *TestCommunication) UnSubscribe(topic communication.ChainBridgeMessageType, sessionID string) {
	delete(ts.Subscriptions, string(topic)+sessionID)
}

func (ts *TestCommunication) ReceiveMessage(msg *communication.WrappedMessage, topic communication.ChainBridgeMessageType, sessionID string) {
	ts.Subscriptions[string(topic)+sessionID] <- msg
}
