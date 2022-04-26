// Copyright 2021 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/ChainSafe/chainbridge-core/tss"
	"github.com/ChainSafe/chainbridge-core/tss/signing"
	"github.com/libp2p/go-libp2p-core/host"

	"github.com/ChainSafe/chainbridge-core/chains/evm/executor/proposal"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ethereum/go-ethereum/common"
	ethereumTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type ChainClient interface {
	RelayerAddress() common.Address
	CallContract(ctx context.Context, callArgs map[string]interface{}, blockNumber *big.Int) ([]byte, error)
	SubscribePendingTransactions(ctx context.Context, ch chan<- common.Hash) (*rpc.ClientSubscription, error)
	TransactionByHash(ctx context.Context, hash common.Hash) (tx *ethereumTypes.Transaction, isPending bool, err error)
	calls.ContractCallerDispatcher
}

type MessageHandler interface {
	HandleMessage(m *message.Message) (*proposal.Proposal, error)
}

type BridgeContract interface {
	ProposalStatus(p *proposal.Proposal) (message.ProposalStatus, error)
}

type Executor struct {
	host    host.Host
	comm    communication.Communication
	fetcher signing.SaveDataFetcher
	bridge  BridgeContract
	mh      MessageHandler
}

func NewExecutor(mh MessageHandler, client ChainClient, bridgeContract BridgeContract) *Executor {
	return &Executor{}
}

func (e *Executor) Execute(m *message.Message) error {
	prop, err := e.mh.HandleMessage(m)
	if err != nil {
		return err
	}

	ps, err := e.bridge.ProposalStatus(prop)
	if err != nil {
		return err
	}
	if ps.Status == message.ProposalStatusExecuted {
		return nil
	}

	signing, err := signing.NewSigning(
		big.NewInt(0),
		fmt.Sprintf("%d-%d", m.Destination, m.DepositNonce),
		e.host,
		e.comm,
		e.fetcher)
	if err != nil {
		return err
	}
	coordinator := tss.NewCoordinator(e.host, signing, e.comm)
	sigChn := make(chan interface{})
	statusChn := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())
	go coordinator.Execute(ctx, sigChn, statusChn)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case sig := <-sigChn:
			{
				fmt.Println(sig)
				cancel()
				return nil
			}
		case status := <-statusChn:
			{
				fmt.Println(status)
				cancel()
				return nil
			}
		case <-ticker.C:
			{

				ps, err := e.bridge.ProposalStatus(prop)
				if err != nil {
					continue
				}

				if ps.Status == message.ProposalStatusExecuted {
					cancel()
					return nil
				}
			}
		}
	}
}
