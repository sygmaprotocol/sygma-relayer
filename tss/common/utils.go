package common

import (
	"math/big"

	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p-core/peer"
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
			return peers, err
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
