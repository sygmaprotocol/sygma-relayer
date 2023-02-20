// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package substrate

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/sygma-relayer/chains"

	"github.com/rs/zerolog/log"
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
	ListenToEvents(ctx context.Context, startBlock *big.Int, domainID uint8, blockstore store.BlockStore, msgChan chan []*message.Message)
}

type ProposalExecutor interface {
	Execute(message *message.Message) error
}

func NewSubstrateChain(listener EventListener, writer ProposalExecutor, blockstore *store.BlockStore, config *SubstrateConfig, executor BatchProposalExecutor) *SubstrateChain {
	return &SubstrateChain{listener: listener, writer: writer, blockstore: blockstore, config: config, executor: executor}
}

// PollEvents is the goroutine that polls blocks and searches Deposit events in them.
// Events are then sent to eventsChan.
func (c *SubstrateChain) PollEvents(ctx context.Context, sysErr chan<- error, msgChan chan []*message.Message) {
	log.Info().Msg("Polling Blocks...")

	startBlock, err := c.blockstore.GetStartBlock(
		*c.config.GeneralChainConfig.Id,
		c.config.StartBlock,
		c.config.GeneralChainConfig.LatestBlock,
		c.config.GeneralChainConfig.FreshStart,
	)

	if err != nil {
		sysErr <- fmt.Errorf("error %w on getting last stored block", err)
		return
	}
	startBlock = chains.CalculateStartingBlock(startBlock, c.config.BlockInterval)

	go c.listener.ListenToEvents(ctx, startBlock, c.DomainID(), *c.blockstore, msgChan)
}

func (c *SubstrateChain) Write(msgs []*message.Message) {
	err := c.executor.Execute(msgs)
	if err != nil {
		log.Err(err).Msgf("error writing messages %+v on network %d", msgs, c.DomainID())
	}
}

func (c *SubstrateChain) DomainID() uint8 {
	return *c.config.GeneralChainConfig.Id
}
