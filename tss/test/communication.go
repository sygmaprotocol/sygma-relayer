package tsstest

import (
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

type Receiver interface {
	ReceiveMessage(msg *common.WrappedMessage, topic common.ChainBridgeMessageType, sessionID string)
}

type TestCommunication struct {
	Host               host.Host
	Subscriptions      map[string]chan *common.WrappedMessage
	PeerCommunications map[string]Receiver
}

func (tc *TestCommunication) Broadcast(
	peers peer.IDSlice,
	msg []byte,
	msgType common.ChainBridgeMessageType,
	sessionID string,
) {
	wMsg := common.WrappedMessage{
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
	topic common.ChainBridgeMessageType,
	sessionID string,
	channel chan *common.WrappedMessage,
) {
	ts.Subscriptions[string(topic)+sessionID] = channel
}

func (ts *TestCommunication) CancelSubscribe(topic common.ChainBridgeMessageType, sessionID string) {
	delete(ts.Subscriptions, string(topic)+sessionID)
}

func (ts *TestCommunication) ReceiveMessage(msg *common.WrappedMessage, topic common.ChainBridgeMessageType, sessionID string) {
	ts.Subscriptions[string(topic)+sessionID] <- msg
}
