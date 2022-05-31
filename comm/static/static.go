package static

import (
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

type CoordinatorElector struct {
	host host.Host
}

func NewStaticCommunicationCoordinator(h host.Host) CoordinatorElector {
	return CoordinatorElector{host: h}
}

func (s *CoordinatorElector) Coordinator(sessionID string) (peer.ID, error) {
	peers := s.host.Peerstore().Peers()
	return common.SortPeersForSession(peers, sessionID)[0].ID, nil
}
