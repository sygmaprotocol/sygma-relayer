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
	// Broadcast sends message to provided peers
	Broadcast(peers peer.IDSlice, msg []byte, msgType ChainBridgeMessageType, sessionID string)
	// Subscribe subscribes provided channel to a specific message type for a provided session
	// Returns SubscriptionID - unique identifier of created subscription that is used to unsubscribe from subscription
	Subscribe(sessionID string, msgType ChainBridgeMessageType, channel chan *WrappedMessage) SubscriptionID
	// UnSubscribe unsuscribes from subscription defined by provided SubscriptionID
	UnSubscribe(subID SubscriptionID)
}
