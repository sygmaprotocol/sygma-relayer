// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package elector

import (
	"context"

	"github.com/ChainSafe/sygma-relayer/tss/common"
	"github.com/libp2p/go-libp2p/core/peer"
)

type staticCoordinatorElector struct {
	sessionID string
}

func NewCoordinatorElector(sessionID string) CoordinatorElector {
	return &staticCoordinatorElector{sessionID: sessionID}
}

func (s *staticCoordinatorElector) Coordinator(ctx context.Context, peers peer.IDSlice) (peer.ID, error) {
	sortedPeers := common.SortPeersForSession(peers, s.sessionID)
	if len(sortedPeers) == 0 {
		return peer.ID(""), nil
	}
	return common.SortPeersForSession(peers, s.sessionID)[0].ID, nil
}
