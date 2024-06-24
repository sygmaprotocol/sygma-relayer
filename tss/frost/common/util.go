// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package common

import (
	"sort"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/taurusgroup/multi-party-sig/pkg/party"
)

func PartyIDSFromPeers(peers peer.IDSlice) []party.ID {
	peerSet := mapset.NewSet[peer.ID](peers...)
	peerSlice := peerSet.ToSlice()

	idSlice := make([]party.ID, len(peerSlice))
	for i, peer := range peerSlice {
		idSlice[i] = party.ID(peer.String())
	}

	// Sort the idSlice
	sort.Slice(idSlice, func(i, j int) bool {
		return idSlice[i] < idSlice[j]
	})

	return idSlice
}
