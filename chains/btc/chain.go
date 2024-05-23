// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package btc

import (
	"context"
	"math/big"

	"github.com/btcsuite/btcd/rpcclient"

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
	ListenToEvents(ctx context.Context, startBlock *big.Int)
}
type BtcChain struct {
	connection *rpcclient.Client
	listener   EventListener
	blockstore *store.BlockStore
	config     *BtcConfig
	startBlock *big.Int
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

// PollEvents is the goroutine that polls blocks and searches Deposit events in them.
// Events are then sent to eventsChan.
func (c *BtcChain) PollEvents(ctx context.Context) {
	c.logger.Info().Str("startBlock", c.startBlock.String()).Msg("Polling Blocks...")
	go c.listener.ListenToEvents(ctx, c.startBlock)
}

func (c *BtcChain) DomainID() uint8 {
	return *c.config.GeneralChainConfig.Id
}
