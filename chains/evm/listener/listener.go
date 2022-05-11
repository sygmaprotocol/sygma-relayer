// Copyright 2021 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/chainbridge-core/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/rs/zerolog/log"
)

type DepositHandler interface {
	HandleDeposit(sourceID, destID uint8, nonce uint64, resourceID types.ResourceID, calldata, handlerResponse []byte) (*message.Message, error)
}
type ChainClient interface {
	LatestBlock() (*big.Int, error)
	CallContract(ctx context.Context, callArgs map[string]interface{}, blockNumber *big.Int) ([]byte, error)
}

type EventListener interface {
	FetchDeposits(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]*events.Deposit, error)
	FetchKeygenEvents(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error)
	FetchRefreshEvents(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error)
}

type EVMListener struct {
	client         ChainClient
	eventListener  EventListener
	depositHandler DepositHandler
	bridgeAddress  common.Address
	domainID       uint8

	blockstore         *store.BlockStore
	blockRetryInterval time.Duration
	blockConfirmations *big.Int
}

// NewEVMListener creates an EVMListener that listens to deposit events on chain
// and calls event handler when one occurs
func NewEVMListener(client ChainClient, eventListener EventListener, depositHandler DepositHandler, bridgeAddress common.Address) *EVMListener {
	return &EVMListener{client: client, eventListener: eventListener, depositHandler: depositHandler, bridgeAddress: bridgeAddress}
}

// ListenToEvents goes block by block of a network and executes event handlers that are
// configured for the listener.
func (l *EVMListener) ListenToEvents(ctx context.Context, startBlock *big.Int, msgChan chan *message.Message, errChn chan<- error) {
	block := startBlock
	for {
		select {
		case <-ctx.Done():
			return
		default:
			block, err := l.nextBlock(block)
			if err != nil {
				log.Err(err).Msgf("Unable to fetch next block because of: %s", err)
				time.Sleep(l.blockRetryInterval)
				continue
			}

			//Write to block store. Not a critical operation, no need to retry
			err = l.blockstore.StoreBlock(block, l.domainID)
			if err != nil {
				log.Error().Str("block", block.String()).Err(err).Msg("Failed to write latest block to blockstore")
			}
		}
	}
}

func (l *EVMListener) nextBlock(previousBlock *big.Int) (*big.Int, error) {
	head, err := l.client.LatestBlock()
	if err != nil {
		return nil, err
	}
	if previousBlock == nil {
		return head, nil
	}

	nextBlock := previousBlock.Add(previousBlock, big.NewInt(1))
	if big.NewInt(0).Sub(head, nextBlock).Cmp(l.blockConfirmations) == -1 {
		return nil, fmt.Errorf("block %s difference from head %s less than configured block confirmations %s", nextBlock, head, l.blockConfirmations)
	}

	return nextBlock, nil
}

/*
func (l *EVMListener) handleDeposits(deposits []*events.Deposit, block *big.Int, domainID uint8, ch chan *message.Message) {
	for _, d := range deposits {
		log.Debug().Msgf("Deposit log found from sender: %s in block: %s with  destinationDomainId: %v, resourceID: %s, depositNonce: %v", d.SenderAddress, block.String(), d.DestinationDomainID, d.ResourceID, d.DepositNonce)

		m, err := l.eventHandler.HandleEvent(domainID, d.DestinationDomainID, d.DepositNonce, d.ResourceID, d.Data, d.HandlerResponse)
		if err != nil {
			log.Error().Str("block", block.String()).Uint8("domainID", domainID).Msgf("%v", err)
			continueblock delay
		}

		log.Debug().Msgf("Resolved message %+v in block %s", m, block.String())
		ch <- m
	}
}
*/
