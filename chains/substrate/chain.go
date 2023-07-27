// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package substrate

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type BatchProposalExecutor interface {
	Execute(ctx context.Context, msgs []*message.Message) error
}

type SubstrateChain struct {
	client     *client.SubstrateClient
	listener   EventListener
	writer     ProposalExecutor
	blockstore *store.BlockStore
	config     *SubstrateConfig
	executor   BatchProposalExecutor
	logger     zerolog.Logger
}

type EventListener interface {
	ListenToEvents(ctx context.Context, startBlock *big.Int, domainID uint8, blockstore store.BlockStore, msgChan chan []*message.Message)
}

type ProposalExecutor interface {
	Execute(ctx context.Context, message *message.Message) error
}

func NewSubstrateChain(client *client.SubstrateClient, listener EventListener, writer ProposalExecutor, blockstore *store.BlockStore, config *SubstrateConfig, executor BatchProposalExecutor) *SubstrateChain {
	return &SubstrateChain{
		client:     client,
		listener:   listener,
		writer:     writer,
		blockstore: blockstore,
		config:     config,
		executor:   executor,
		logger:     log.With().Str("domainID", string(*config.GeneralChainConfig.Id)).Logger()}
}

// PollEvents is the goroutine that polls blocks and searches Deposit events in them.
// Events are then sent to eventsChan.
func (c *SubstrateChain) PollEvents(ctx context.Context, sysErr chan<- error, msgChan chan []*message.Message) {
	c.logger.Info().Msg("Polling Blocks...")

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
	// start from latest
	if startBlock == nil {
		head, err := c.client.LatestBlock()
		if err != nil {
			sysErr <- fmt.Errorf("error %w on getting latest block for domain %d", err, c.DomainID())
			return
		}
		startBlock = head
	}

	startBlock, err = chains.CalculateStartingBlock(startBlock, c.config.BlockInterval)
	if err != nil {
		sysErr <- fmt.Errorf("error %w on CalculateStartingBlock domain %d", err, c.DomainID())
		return
	}

	c.logger.Info().Msgf("Starting block: %s", startBlock.String())

	go c.listener.ListenToEvents(ctx, startBlock, c.DomainID(), *c.blockstore, msgChan)
}

func (c *SubstrateChain) Write(ctx context.Context, msgs []*message.Message) error {
	err := c.executor.Execute(ctx, msgs)
	if err != nil {
		c.logger.Err(err).Msgf("error writing messages %+v on network %d", msgs, c.DomainID())
		return err
	}

	return nil
}

func (c *SubstrateChain) DomainID() uint8 {
	return *c.config.GeneralChainConfig.Id
}
