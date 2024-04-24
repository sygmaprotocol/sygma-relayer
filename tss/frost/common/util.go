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
	sort.Sort(peers)
	peerSet := mapset.NewSet[peer.ID](peers...)
	idSlice := make([]party.ID, len(peerSet.ToSlice()))
	for i, peer := range peerSet.ToSlice() {
		idSlice[i] = party.ID(peer.Pretty())
	}
	return idSlice
}
