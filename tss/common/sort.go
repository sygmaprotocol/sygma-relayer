package common

import (
	"encoding/binary"
	"reflect"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

type PeerMsg struct {
	ID        peer.ID
	SessionID string
}

type SortablePeerSlice []PeerMsg

func (sps SortablePeerSlice) Len() int {
	return len(sps)
}

func (sps SortablePeerSlice) Swap(i, j int) {
	reflect.Swapper(sps)(i, j)
}

func (sps SortablePeerSlice) Less(i, j int) bool {
	crypto.Keccak256(append([]byte(sps[i].ID.Pretty()), []byte(sps[i].SessionID)...))
	iHash := crypto.Keccak256(append([]byte(sps[i].ID.Pretty()), []byte(sps[i].SessionID)...))
	jHash := crypto.Keccak256(append([]byte(sps[j].ID.Pretty()), []byte(sps[j].SessionID)...))
	return binary.BigEndian.Uint64(iHash) > binary.BigEndian.Uint64(jHash)
}
