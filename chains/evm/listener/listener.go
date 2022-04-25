// Copyright 2021 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"context"
	"math/big"
	"time"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/chainbridge-core/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/rs/zerolog/log"
)

type EventHandler interface {
	HandleEvent(sourceID, destID uint8, nonce uint64, resourceID types.ResourceID, calldata, handlerResponse []byte) (*message.Message, error)
}
type ChainClient interface {
	LatestBlock() (*big.Int, error)
	CallContract(ctx context.Context, callArgs map[string]interface{}, blockNumber *big.Int) ([]byte, error)
}

type EventListener interface {
	FetchDeposits(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]*events.Deposit, error)
}

type EVMListener struct {
	client        ChainClient
	eventListener EventListener
	eventHandler  EventHandler
	bridgeAddress common.Address
}

// NewEVMListener creates an EVMListener that listens to deposit events on chain
// and calls event handler when one occurs
func NewEVMListener(client ChainClient, eventListener EventListener, handler EventHandler, bridgeAddress common.Address) *EVMListener {
	return &EVMListener{client: client, eventListener: eventListener, eventHandler: handler, bridgeAddress: bridgeAddress}
}

func (l *EVMListener) ListenToEvents(
	startBlock, blockDelay *big.Int,
	blockRetryInterval time.Duration,
	domainID uint8,
	blockstore *store.BlockStore,
	stopChn <-chan struct{},
	errChn chan<- error,
) <-chan *message.Message {
	ch := make(chan *message.Message)
	go func() {
		for {
			select {
			case <-stopChn:
				return
			default:
				head, err := l.client.LatestBlock()
				if err != nil {
					log.Error().Err(err).Msg("Unable to get latest block")
					time.Sleep(blockRetryInterval)
					continue
				}

				if startBlock == nil {
					startBlock = head
				}

				// Sleep if the difference is less than blockDelay; (latest - current) < BlockDelay
				if big.NewInt(0).Sub(head, startBlock).Cmp(blockDelay) == -1 {
					time.Sleep(blockRetryInterval)
					continue
				}

				if startBlock.Int64()%20 == 0 {
					// Logging process every 20 bocks to exclude spam
					log.Debug().Str("block", startBlock.String()).Uint8("domainID", domainID).Msg("Queried block for briding events")
				}

				deposits, err := l.eventListener.FetchDeposits(context.Background(), l.bridgeAddress, startBlock, startBlock)
				if err != nil {
					// Filtering logs error really can appear only on wrong configuration or temporary network problem
					// so i do no see any reason to break execution
					log.Error().Err(err).Str("DomainID", string(domainID)).Msgf("Unable to filter logs")
					continue
				}
				l.handleDeposits(deposits, startBlock, domainID, ch)

				//Write to block store. Not a critical operation, no need to retry
				err = blockstore.StoreBlock(startBlock, domainID)
				if err != nil {
					log.Error().Str("block", startBlock.String()).Err(err).Msg("Failed to write latest block to blockstore")
				}

				// Goto next block
				startBlock.Add(startBlock, big.NewInt(1))
			}
		}
	}()

	return ch
}

func (l *EVMListener) handleDeposits(deposits []*events.Deposit, block *big.Int, domainID uint8, ch chan *message.Message) {
	for _, d := range deposits {
		log.Debug().Msgf("Deposit log found from sender: %s in block: %s with  destinationDomainId: %v, resourceID: %s, depositNonce: %v", d.SenderAddress, block.String(), eventLog.DestinationDomainID, eventLog.ResourceID, eventLog.DepositNonce)

		m, err := l.eventHandler.HandleEvent(domainID, d.DestinationDomainID, d.DepositNonce, d.ResourceID, d.Data, d.HandlerResponse)
		if err != nil {
			log.Error().Str("block", block.String()).Uint8("domainID", domainID).Msgf("%v", err)
			continue
		}

		log.Debug().Msgf("Resolved message %+v in block %s", m, block.String())
		ch <- m
	}
}
