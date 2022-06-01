package static

import (
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/libp2p/go-libp2p-core/peer"
)

type CoordinatorElector struct {
	sessionID string
}

func NewCoordinatorElector(sessionID string) CoordinatorElector {
	return CoordinatorElector{sessionID: sessionID}
}

func (s CoordinatorElector) Coordinator(peers peer.IDSlice) (peer.ID, error) {
	return common.SortPeersForSession(peers, s.sessionID)[0].ID, nil
}
