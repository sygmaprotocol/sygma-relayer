// Copyright 2021 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"context"
	"math/big"
	"time"

	"github.com/ChainSafe/sygma-relayer/chains/btc"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sygmaprotocol/sygma-core/store"
)

type EventHandler interface {
	HandleEvents(startBlock *big.Int) error
}

type BtcListener struct {
	conn rpcclient.Client

	eventHandlers      []EventHandler
	blockRetryInterval time.Duration

	log      zerolog.Logger
	domainID uint8
}

// NewBtcListener creates an BtcListener that listens to deposit events on chain
// and calls event handler when one occurs
func NewBtcListener(connection rpcclient.Client, eventHandlers []EventHandler, config *btc.BtcConfig, domainID uint8) *BtcListener {
	return &BtcListener{
		log:                log.With().Uint8("domainID", *config.GeneralChainConfig.Id).Logger(),
		conn:               connection,
		eventHandlers:      eventHandlers,
		blockRetryInterval: 10,
		domainID:           domainID,
	}
}

// ListenToEvents goes block by block of a network and executes event handlers that are
// configured for the listener.
func (l *BtcListener) ListenToEvents(ctx context.Context, startBlock *big.Int, domainID uint8, blockstore store.BlockStore) {
loop:
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get the hash of the most recent block
			bestBlockHash, err := l.conn.GetBestBlockHash()
			if err != nil {
				l.log.Warn().Err(err).Msg("Unable to get latest block")
				time.Sleep(l.blockRetryInterval)
				return
			}

			// Fetch the most recent block in verbose mode to get additional information including height
			block, err := l.conn.GetBlockVerboseTx(bestBlockHash)
			if err != nil {
				l.log.Warn().Err(err).Msg("Unable to get latest block")
				time.Sleep(l.blockRetryInterval)
				continue
			}
			head := big.NewInt(block.Height)
			if startBlock == nil {
				startBlock = head
			}

			// Sleep if the difference is less than needed block confirmations; (latest - current) < BlockDelay
			if startBlock.Cmp(head) == 1 {
				time.Sleep(l.blockRetryInterval)
				continue
			}

			for _, handler := range l.eventHandlers {
				err := handler.HandleEvents(startBlock)
				if err != nil {
					l.log.Warn().Err(err).Msgf("Unable to handle events")
					continue loop
				}
			}

			//Write to block store. Not a critical operation, no need to retry
			err = blockstore.StoreBlock(startBlock, l.domainID)
			if err != nil {
				l.log.Error().Str("block", startBlock.String()).Err(err).Msg("Failed to write latest block to blockstore")
			}

			startBlock.Add(startBlock, big.NewInt(1))
		}
	}
}
