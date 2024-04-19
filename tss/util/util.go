package util

import (
	"sort"

	"github.com/libp2p/go-libp2p/core/peer"
)

func SortPeersForSession(peers []peer.ID, sessionID string) SortablePeerSlice {
	sortedPeers := make(SortablePeerSlice, len(peers))
	for i, p := range peers {
		pMsg := PeerMsg{
			ID:        p,
			SessionID: sessionID,
		}
		sortedPeers[i] = pMsg
	}
	sort.Sort(sortedPeers)
	return sortedPeers
}

func IsParticipant(peer peer.ID, peers peer.IDSlice) bool {
	for _, p := range peers {
		if p.Pretty() == peer.Pretty() {
			return true
		}
	}

	return false
}
