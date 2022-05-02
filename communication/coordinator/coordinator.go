package coordinator

import "github.com/libp2p/go-libp2p-core/peer"

type Coordinator interface {
	// GetCoordinatorForSession calculates static coordinator for provided session
	GetCoordinatorForSession(sessionID string) peer.ID
	// IsCoordinatorForSession checks if this host is static coordinator for provided session
	IsCoordinatorForSession(sessionID string) bool
	// StartBullyCoordination starts bully process to determine dynamic coordinator
	// When coordinator is determined it will be sent to coordinatorChan, if error occurs it will be sent to errChan
	StartBullyCoordination(excludedPeers peer.IDSlice, coordinatorChan chan peer.ID, errChan chan error)
}
