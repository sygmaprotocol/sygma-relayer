package common

import (
	"math/big"
	"sort"

	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p-core/peer"
	"golang.org/x/exp/slices"
)

func CreatePartyID(peerID string) *tss.PartyID {
	key := big.NewInt(0).SetBytes([]byte(peerID))
	return tss.NewPartyID(peerID, peerID, key)
}

func IsParticipant(party *tss.PartyID, parties tss.SortedPartyIDs) bool {
	for _, existingParty := range parties {
		if party.Id == existingParty.Id {
			return true
		}
	}

	return false
}

func PeersFromParties(parties []*tss.PartyID) ([]peer.ID, error) {
	peers := make([]peer.ID, len(parties))
	for i, party := range parties {
		peerID, err := peer.Decode(party.Id)
		if err != nil {
			return nil, err
		}

		peers[i] = peerID
	}

	return peers, nil
}

func PartiesFromPeers(peers peer.IDSlice) tss.SortedPartyIDs {
	unsortedParties := make(tss.UnSortedPartyIDs, len(peers))

	for i, peer := range peers {
		unsortedParties[i] = CreatePartyID(peer.String())
	}

	return tss.SortPartyIDs(unsortedParties)
}

func PeersFromIDS(peerIDS []string) ([]peer.ID, error) {
	peers := make([]peer.ID, len(peerIDS))

	for i, id := range peerIDS {
		peerID, err := peer.Decode(id)
		if err != nil {
			return nil, err
		}

		peers[i] = peerID
	}

	return peers, nil
}

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

func ExcludePeers(peers peer.IDSlice, excludedPeers peer.IDSlice) peer.IDSlice {
	includedPeers := make(peer.IDSlice, 0)
	for _, peer := range peers {
		if slices.Contains(excludedPeers, peer) {
			continue
		}

		includedPeers = append(includedPeers, peer)
	}
	return includedPeers
}
