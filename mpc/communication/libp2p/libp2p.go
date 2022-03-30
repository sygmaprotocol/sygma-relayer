package libp2p

import (
	"github.com/rs/zerolog"
	"sync"
)

type Libp2pCommunication struct {
	h                   host.Host
	fullAddressAsString string
	protocolID          protocol.ID
	streamManager       *StreamManager
	logger              zerolog.Logger
	subscribers         map[ChainBridgeMessageType]*MessageIDSubscriber
	subscriberLocker    *sync.Mutex
}

func NewCommunication() Libp2pCommunication {
	return Libp2pCommunication{}
}

func (c *Libp2pCommunication) Broadcast(peers peer.IDSlice, msg []byte, msgType ChainBridgeMessageType, sessionID string) {

}
