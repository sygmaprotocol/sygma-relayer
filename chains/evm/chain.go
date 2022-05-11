// Copyright 2021 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmgaspricer"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/signAndSend"
	"github.com/ChainSafe/chainbridge-core/chains/evm/executor"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/listener"
	"github.com/ChainSafe/chainbridge-core/config/chain"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

type EventListener interface {
	ListenToEvents(ctx context.Context, startBlock *big.Int, msgChan chan *message.Message, errChan chan<- error)
}

type ProposalExecutor interface {
	Execute(message *message.Message) error
}

// EVMChain is struct that aggregates all data required for
type EVMChain struct {
	listener   EventListener
	writer     ProposalExecutor
	blockstore *store.BlockStore
	config     *chain.EVMConfig
}

// SetupDefaultEVMChain sets up an EVMChain with all supported handlers configured
func SetupDefaultEVMChain(rawConfig map[string]interface{}, txFabric calls.TxFabric, blockstore *store.BlockStore) (*EVMChain, error) {
	config, err := chain.NewEVMConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client, err := evmclient.NewEVMClient(config)
	if err != nil {
		return nil, err
	}

	bridgeAddress := common.HexToAddress(config.Bridge)
	gasPricer := evmgaspricer.NewLondonGasPriceClient(client, nil)
	t := signAndSend.NewSignAndSendTransactor(txFabric, gasPricer, client)
	bridgeContract := bridge.NewBridgeContract(client, common.HexToAddress(config.Bridge), t)

	depositHandler := listener.NewETHDepositHandler(*bridgeContract)
	depositHandler.RegisterDepositHandler(config.Erc20Handler, listener.Erc20DepositHandler)
	depositHandler.RegisterDepositHandler(config.Erc721Handler, listener.Erc721DepositHandler)
	depositHandler.RegisterDepositHandler(config.GenericHandler, listener.GenericDepositHandler)
	eventListener := events.NewListener(client)
	eventHandlers := make([]listener.EventHandler, 0)
	eventHandlers = append(eventHandlers, listener.NewDepositEventHandler(eventListener, depositHandler, bridgeAddress, *config.GeneralChainConfig.Id))
	eventHandlers = append(eventHandlers, listener.NewKeygenEventHandler(eventListener, bridgeAddress))
	eventHandlers = append(eventHandlers, listener.NewRefreshEventHandler(eventListener, bridgeAddress))
	evmListener := listener.NewEVMListener(client, eventHandlers, blockstore, config)

	mh := executor.NewEVMMessageHandler(*bridgeContract)
	mh.RegisterMessageHandler(config.Erc20Handler, executor.ERC20MessageHandler)
	mh.RegisterMessageHandler(config.Erc721Handler, executor.ERC721MessageHandler)
	mh.RegisterMessageHandler(config.GenericHandler, executor.GenericMessageHandler)

	return NewEVMChain(evmListener, nil, blockstore, config), nil
}

func NewEVMChain(listener EventListener, writer ProposalExecutor, blockstore *store.BlockStore, config *chain.EVMConfig) *EVMChain {
	return &EVMChain{listener: listener, writer: writer, blockstore: blockstore, config: config}
}

// PollEvents is the goroutine that polls blocks and searches Deposit events in them.
// Events are then sent to eventsChan.
func (c *EVMChain) PollEvents(ctx context.Context, sysErr chan<- error, msgChan chan *message.Message) {
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

	go c.listener.ListenToEvents(ctx, startBlock, msgChan, sysErr)
}

func (c *EVMChain) Write(msg *message.Message) error {
	return c.writer.Execute(msg)
}

func (c *EVMChain) DomainID() uint8 {
	return *c.config.GeneralChainConfig.Id
}
