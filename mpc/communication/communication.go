package communication

type Communication interface {
	Broadcast(peers peer.IDSlice, msg []byte, msgType ChainBridgeMessageType, sessionID string) // TODO: message type
	BroadcastToSavedPeers(msg []byte, msgType ChainBridgeMessageType, sessionID string)
	ReleaseStream(sessionID string)
	Subscribe(topic ChainBridgeMessageType, sessionID string, channel chan *WrappedMessage)
	CancelSubscribe(topic ChainBridgeMessageType, sessionID string)
}
