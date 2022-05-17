package static

import (
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

type CommunicationCoordinator struct {
	host host.Host
}

func NewStaticCommunicationCoordinator(h host.Host) CommunicationCoordinator {
	return CommunicationCoordinator{host: h}
}

func (s *CommunicationCoordinator) GetCoordinator(sessionID string) (peer.ID, error) {
	peers := s.host.Peerstore().Peers()
	return common.SortPeersForSession(peers, sessionID)[0].ID, nil
}
