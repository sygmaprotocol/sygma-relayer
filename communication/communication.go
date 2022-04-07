package communication

import (
	"github.com/ChainSafe/chainbridge-core/communication/p2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog"
)

// WrappedMessage is a message with type in it
type WrappedMessage struct {
	MessageType p2p.ChainBridgeMessageType `json:"message_type"`
	SessionID   SessionID                  `json:"message_id"`
	Payload     []byte                     `json:"payload"`
	From        peer.ID                    `json:"from"`
}

//
type SessionID string

func (sid SessionID) MarshalZerologObject(e *zerolog.Event) {
	e.Str("sessionID", string(sid))
}

//
type SubscriptionID string

func (sid SubscriptionID) MarshalZerologObject(e *zerolog.Event) {
	e.Str("subscriptionID", string(sid))
}

// Communication
type Communication interface {
	Broadcast(peers peer.IDSlice, msg []byte, msgType p2p.ChainBridgeMessageType, sID SessionID)
	ReleaseStream(sID SessionID)
	Subscribe(msgType p2p.ChainBridgeMessageType, sID SessionID, channel chan *WrappedMessage) SubscriptionID
	UnSubscribe(subID SubscriptionID)
}
