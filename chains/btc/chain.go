// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package btc

import (
	"context"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/rpcclient"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/sygma-relayer/chains"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type BatchProposalExecutor interface {
	Execute(msgs []*message.Message) error
}
type EventListener interface {
	ListenToEvents(ctx context.Context, startBlock *big.Int, domainID uint8, blockstore store.BlockStore, msgChan chan []*message.Message)
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

	fmt.Println("After all checks, before creating BtcChain instance")
	return &BtcChain{
		connection: connection,
		listener:   listener,
		blockstore: blockstore,
		config:     config,
		logger:     log.With().Str("domainID", string(*config.GeneralChainConfig.Id)).Logger()}
}

func (c *BtcChain) Write(msgs []*message.Message) error {
	return nil
}

func (c *BtcChain) PollEvents(ctx context.Context, sysErr chan<- error, msgChan chan []*message.Message) {
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
		// Get the hash of the most recent block
		bestBlockHash, err := c.connection.GetBestBlockHash()
		if err != nil {
			sysErr <- fmt.Errorf("error %w on getting latest block for domain %d", err, c.DomainID())
			return
		}

		// Fetch the most recent block
		head, err := c.connection.GetBlockVerboseTx(bestBlockHash)
		if err != nil {
			sysErr <- fmt.Errorf("error %w on getting latest block for domain %d", err, c.DomainID())
			return
		}
		startBlock = new(big.Int).SetInt64(head.Height)
	}

	startBlock, err = chains.CalculateStartingBlock(startBlock, c.config.BlockInterval)
	if err != nil {
		sysErr <- fmt.Errorf("error %w on CalculateStartingBlock domain %d", err, c.DomainID())
		return
	}

	c.logger.Info().Msgf("Starting block: %s", startBlock.String())

	go c.listener.ListenToEvents(ctx, startBlock, c.DomainID(), *c.blockstore, msgChan)
}

func (c *BtcChain) DomainID() uint8 {
	return *c.config.GeneralChainConfig.Id
}
