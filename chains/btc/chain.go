// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package btc

import (
	"context"
	"math/big"

	"github.com/btcsuite/btcd/rpcclient"

	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
	"github.com/sygmaprotocol/sygma-core/store"
)

type BatchProposalExecutor interface {
	Execute(msgs []*message.Message) error
}
type EventListener interface {
	ListenToEvents(ctx context.Context, startBlock *big.Int, domainID uint8, blockstore store.BlockStore)
}
type BtcChain struct {
	connection *rpcclient.Client
	listener   EventListener
	blockstore *store.BlockStore
	config     *BtcConfig
	logger     zerolog.Logger
}

func NewBtcChain(
	connection *rpcclient.Client, listener EventListener,
	blockstore *store.BlockStore, config *BtcConfig,
) *BtcChain {
	return &BtcChain{
		connection: connection,
		listener:   listener,
		blockstore: blockstore,
		config:     config,
		logger:     log.With().Str("domainID", string(*config.GeneralChainConfig.Id)).Logger()}
}

func (c *BtcChain) Write(msgs []*proposal.Proposal) error {
	return nil
}

// remove this after
func (c *BtcChain) ReceiveMessage(m *message.Message)/* *proposal.Proposal, error */ {}

func (c *BtcChain) PollEvents(ctx context.Context) {
	c.logger.Info().Msg("Polling Blocks...")

	startBlock, err := c.blockstore.GetStartBlock(
		*c.config.GeneralChainConfig.Id,
		c.config.StartBlock,
		c.config.GeneralChainConfig.LatestBlock,
		c.config.GeneralChainConfig.FreshStart,
	)
	if err != nil {
		return
	}

	// start from latest
	if startBlock == nil {
		// Get the hash of the most recent block
		bestBlockHash, err := c.connection.GetBestBlockHash()
		if err != nil {
			return
		}

		// Fetch the most recent block
		head, err := c.connection.GetBlockVerboseTx(bestBlockHash)
		if err != nil {
			return
		}
		startBlock = new(big.Int).SetInt64(head.Height)
	}

	startBlock, err = chains.CalculateStartingBlock(startBlock, c.config.BlockInterval)
	if err != nil {
		return
	}

	c.logger.Info().Msgf("Starting block: %s", startBlock.String())

	go c.listener.ListenToEvents(ctx, startBlock, c.DomainID(), *c.blockstore)
}

func (c *BtcChain) DomainID() uint8 {
	return *c.config.GeneralChainConfig.Id
}
