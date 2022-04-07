package communication

import (
	"github.com/libp2p/go-libp2p-core/peer"
)

// WrappedMessage is a message with type in it
type WrappedMessage struct {
	MessageType ChainBridgeMessageType `json:"message_type"`
	SessionID   string                 `json:"message_id"`
	Payload     []byte                 `json:"payload"`
	From        peer.ID                `json:"from"`
}

// Communication
type Communication interface {
	Broadcast(peers peer.IDSlice, msg []byte, msgType ChainBridgeMessageType, sessionID string)
	EndSession(sessionID string)
	Subscribe(msgType ChainBridgeMessageType, sessionID string, channel chan *WrappedMessage) string
	UnSubscribe(subscriptionID string)
}
