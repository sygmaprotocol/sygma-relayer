// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/chains/evm"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type BatchProposalExecutor interface {
	Execute(ctx context.Context, msgs []*message.Message) error
}

type EVMChain struct {
	client        *evmclient.EVMClient
	listener      evm.EventListener
	executor      BatchProposalExecutor
	blockstore    *store.BlockStore
	domainID      uint8
	startBlock    *big.Int
	blockInterval *big.Int
	freshStart    bool
	latestBlock   bool
	logger        zerolog.Logger
}

func NewEVMChain(
	client *evmclient.EVMClient, listener evm.EventListener, executor BatchProposalExecutor,
	blockstore *store.BlockStore, domainID uint8, startBlock *big.Int, blockInterval *big.Int,
	freshStart bool, latestBlock bool,
) *EVMChain {
	return &EVMChain{
		client:        client,
		listener:      listener,
		executor:      executor,
		blockstore:    blockstore,
		domainID:      domainID,
		startBlock:    startBlock,
		blockInterval: blockInterval,
		freshStart:    freshStart,
		latestBlock:   latestBlock,
		logger:        log.With().Uint8("domainID", domainID).Logger(),
	}
}

func (c *EVMChain) Write(ctx context.Context, msgs []*message.Message) error {
	err := c.executor.Execute(ctx, msgs)
	if err != nil {
		log.Err(err).Str("domainID", string(c.domainID)).Msgf("error writing messages %+v", msgs)
		return err
	}
	return nil
}

func (c *EVMChain) PollEvents(ctx context.Context, sysErr chan<- error, msgChan chan []*message.Message) {
	c.logger.Info().Msg("Polling Blocks...")

	startBlock, err := c.blockstore.GetStartBlock(
		c.domainID,
		c.startBlock,
		c.latestBlock,
		c.freshStart,
	)
	if err != nil {
		sysErr <- fmt.Errorf("error %w on getting last stored block for domain %d", err, c.domainID)
		return
	}

	// start from latest
	if startBlock == nil {
		head, err := c.client.LatestBlock()
		if err != nil {
			sysErr <- fmt.Errorf("error %w on getting latest block for domain %d", err, c.domainID)
			return
		}
		startBlock = head
	}

	startBlock, err = chains.CalculateStartingBlock(startBlock, c.blockInterval)
	if err != nil {
		sysErr <- fmt.Errorf("error %w on CalculateStartingBlock domain %d", err, c.domainID)
		return
	}

	c.logger.Info().Msgf("Starting block: %s", startBlock.String())

	go c.listener.ListenToEvents(ctx, startBlock, msgChan, sysErr)
}

func (c *EVMChain) DomainID() uint8 {
	return c.domainID
}
