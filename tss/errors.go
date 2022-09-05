// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package tss

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
)

type CoordinatorError struct {
	Peer peer.ID
}

func (ce *CoordinatorError) Error() string {
	return fmt.Sprintf("coordinator %s non-responsive", ce.Peer.Pretty())
}

type SubsetError struct {
	Peer peer.ID
}

func (se *SubsetError) Error() string {
	return fmt.Sprintf("party %s not in signing subset", se.Peer)
}
