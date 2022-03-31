package libp2p

import (
	comm "github.com/ChainSafe/chainbridge-core/mpc/communication"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/rs/zerolog"
	"sync"
)

type Libp2pCommunication struct {
	h                   host.Host
	fullAddressAsString string
	protocolID          protocol.ID
	streamManager       *StreamManager
	logger              zerolog.Logger
	subscribers         map[comm.ChainBridgeMessageType]*SessionSubscriptionManager
	subscriberLocker    *sync.Mutex
}

func NewCommunication() Libp2pCommunication {
	return Libp2pCommunication{}
}

func (c *Libp2pCommunication) Broadcast(peers peer.IDSlice, msg []byte, msgType comm.ChainBridgeMessageType, sessionID string) {

}
