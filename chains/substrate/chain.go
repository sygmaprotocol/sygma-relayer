// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package substrate

import (
	"context"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/store"
)

type BatchProposalExecutor interface {
	Execute(msgs []*message.Message) error
}

type SubstrateChain struct {
	listener   EventListener
	writer     ProposalExecutor
	blockstore *store.BlockStore
	config     *SubstrateConfig
	executor   BatchProposalExecutor
}

type EventListener interface {
	ListenToEvents(ctx context.Context, startBlock *big.Int, msgChan chan []*message.Message, errChan chan<- error)
}

type ProposalExecutor interface {
	Execute(message *message.Message) error
}

func NewSubstrateChain(listener EventListener, writer ProposalExecutor, blockstore *store.BlockStore, config *SubstrateConfig, executor BatchProposalExecutor) *SubstrateChain {
	return &SubstrateChain{listener: listener, writer: writer, blockstore: blockstore, config: config, executor: executor}
}
