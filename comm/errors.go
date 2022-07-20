package comm

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
)

type CommunicationError struct {
	Peer peer.ID
	Err  error
}

func (ce *CommunicationError) Error() string {
	return fmt.Sprintf("failed communicating with peer %s because of: %s", ce.Peer.Pretty(), ce.Err.Error())
}
