package evm

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ChainSafe/sygma/chains/evm/calls/events"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var TestTimeout = time.Minute * 2
var setupTimeout = time.Minute * 30

type Client interface {
	LatestBlock() (*big.Int, error)
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]types.Log, error)
}

func WaitForProposalExecuted(client Client, bridge common.Address) error {
	startBlock, _ := client.LatestBlock()

	query := ethereum.FilterQuery{
		FromBlock: startBlock,
		Addresses: []common.Address{bridge},
		Topics: [][]common.Hash{
			{events.ProposalExecutionSig.GetTopic()},
		},
	}
	timeout := time.After(TestTimeout)
	ch := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, ch)
	if err != nil {
		return err
	}

	defer sub.Unsubscribe()
	for {
		select {
		case <-ch:
			return nil
		case err := <-sub.Err():
			if err != nil {
				return err
			}
		case <-timeout:
			return errors.New("test timed out waiting for proposal execution event")
		}
	}
}

func WaitUntilBridgeReady(client Client, feeHandlerAddress common.Address) error {
	startBlock, _ := client.LatestBlock()
	logs, err := client.FetchEventLogs(context.Background(), feeHandlerAddress, string(events.FeeChangedSig), big.NewInt(1), startBlock)
	if err != nil {
		return err
	}
	if len(logs) > 0 {
		return nil
	}

	query := ethereum.FilterQuery{
		FromBlock: startBlock,
		Addresses: []common.Address{feeHandlerAddress},
		Topics: [][]common.Hash{
			{events.FeeChangedSig.GetTopic()},
		},
	}
	timeout := time.After(setupTimeout)

	ch := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, ch)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	for {
		select {
		case <-ch:
			return nil
		case err := <-sub.Err():
			if err != nil {
				return err
			}
		case <-timeout:
			return errors.New("test timed out waiting for bridge setup")
		}
	}
}
