package tss

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
)

type CoordinatorError struct {
	Coordinator peer.ID
}

func (ce *CoordinatorError) Error() string {
	return fmt.Sprintf("coordinator %s non-responsive", ce.Coordinator.Pretty())
}
