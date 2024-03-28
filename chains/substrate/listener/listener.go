// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"math/big"

	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/parser"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Connection interface {
	GetFinalizedHead() (types.Hash, error)
	GetBlock(blockHash types.Hash) (*types.SignedBlock, error)
	GetBlockHash(blockNumber uint64) (types.Hash, error)
	GetBlockEvents(hash types.Hash) ([]*parser.Event, error)
	UpdateMetatdata() error
}

func FetchEvents(startBlock *big.Int, endBlock *big.Int, conn Connection) ([]*parser.Event, error) {

	evts := make([]*parser.Event, 0)
	for i := new(big.Int).Set(startBlock); i.Cmp(endBlock) == -1; i.Add(i, big.NewInt(1)) {
		hash, err := conn.GetBlockHash(i.Uint64())
		if err != nil {
			return nil, err
		}

		evt, err := conn.GetBlockEvents(hash)
		if err != nil {
			return nil, err
		}
		evts = append(evts, evt...)

	}

	return evts, nil
}
