// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"math/big"
	"strings"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/consts"
	"github.com/ChainSafe/sygma-relayer/chains/substrate"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/parser"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

// ŠTO S fetchEvents????

type EventHandler interface {
	HandleEvents(evts []*parser.Event, msgChan chan []*message.Message) error
}

type ChainConnection interface {
	UpdateMetadata() error
	GetHeaderLatest() (*types.Header, error)
	GetBlockHash(blockNumber uint64) (types.Hash, error)
	GetBlockEvents(hash types.Hash) ([]*parser.Event, error)
	GetFinalizedHead() (types.Hash, error)
	GetBlock(blockHash types.Hash) (*types.SignedBlock, error)
}

type SubstrateListener struct {
	conn ChainConnection
	abi  abi.ABI
	log  zerolog.Logger
}

func NewListener(conn ChainConnection) *SubstrateListener {
	abi, _ := abi.JSON(strings.NewReader(consts.BridgeABI))

	return &SubstrateListener{
		conn: conn,
		abi:  abi,
		log:  log.With().Uint8("domainID", substrate.SubstrateConfig.GeneralChainConfig.Id).Logger(),
	}
}

func (l *SubstrateListener) fetchEvents(startBlock *big.Int, endBlock *big.Int) ([]*parser.Event, error) {
	l.log.Debug().Msgf("Fetching substrate events for block range %s-%s", startBlock, endBlock)

	evts := make([]*parser.Event, 0)
	for i := new(big.Int).Set(startBlock); i.Cmp(endBlock) == -1; i.Add(i, big.NewInt(1)) {
		hash, err := l.conn.GetBlockHash(i.Uint64())
		if err != nil {
			return nil, err
		}

		evt, err := l.conn.GetBlockEvents(hash)
		if err != nil {
			return nil, err
		}
		evts = append(evts, evt...)

	}

	return evts, nil
}
