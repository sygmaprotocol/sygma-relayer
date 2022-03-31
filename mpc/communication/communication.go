package communication

// WrappedMessage is a message with type in it
type WrappedMessage struct {
	MessageType ChainBridgeMessageType `json:"message_type"`
	SessionID   SessionID              `json:"message_id"`
	Payload     []byte                 `json:"payload"`
	From        PeerID                 `json:"from"`
}

//
type ChainBridgeMessageType uint8

//
type SessionID string

//
type PeerID string

//
type SubscriptionID string

// Communication
type Communication interface {
	Broadcast(peers []PeerID, msg []byte, msgType ChainBridgeMessageType, sID SessionID)
	ReleaseStream(sID SessionID)
	Subscribe(msgType ChainBridgeMessageType, sID SessionID, channel chan *WrappedMessage) SubscriptionID
	CancelSubscribe(sID SubscriptionID)
}
