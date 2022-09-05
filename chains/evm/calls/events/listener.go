// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"

	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/consts"
)

type ChainClient interface {
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error)
	WaitAndReturnTxReceipt(h common.Hash) (*ethTypes.Receipt, error)
	LatestBlock() (*big.Int, error)
}

type Listener struct {
	client ChainClient
	abi    abi.ABI
}

func NewListener(client ChainClient) *Listener {
	abi, _ := abi.JSON(strings.NewReader(consts.BridgeABI))
	return &Listener{
		client: client,
		abi:    abi,
	}
}

func (l *Listener) FetchDepositEvent(event RetryEvent, bridgeAddress common.Address, blockConfirmations *big.Int) ([]events.Deposit, error) {
	depositEvents := make([]events.Deposit, 0)
	retryDepositTxHash := common.HexToHash(event.TxHash)
	receipt, err := l.client.WaitAndReturnTxReceipt(retryDepositTxHash)
	if err != nil {
		return depositEvents, fmt.Errorf(
			"unable to fetch logs for retried deposit %s, because of: %+v", retryDepositTxHash.Hex(), err,
		)
	}
	latestBlock, err := l.client.LatestBlock()
	if err != nil {
		return depositEvents, err
	}
	if latestBlock.Cmp(receipt.BlockNumber.Add(receipt.BlockNumber, blockConfirmations)) != 1 {
		return depositEvents, fmt.Errorf(
			"latest block %s higher than receipt block number + block confirmations %s",
			latestBlock,
			receipt.BlockNumber.Add(receipt.BlockNumber, blockConfirmations),
		)
	}

	for _, lg := range receipt.Logs {
		if lg.Address != bridgeAddress {
			continue
		}

		var depositEvent events.Deposit
		err := l.abi.UnpackIntoInterface(&depositEvent, "Deposit", lg.Data)
		if err == nil {
			depositEvents = append(depositEvents, depositEvent)
		}
	}

	return depositEvents, nil
}

func (l *Listener) FetchRetryEvents(ctx context.Context, contractAddress common.Address, startBlock *big.Int, endBlock *big.Int) ([]RetryEvent, error) {
	logs, err := l.client.FetchEventLogs(ctx, contractAddress, string(RetrySig), startBlock, endBlock)
	if err != nil {
		return nil, err
	}

	var retryEvents []RetryEvent
	for _, dl := range logs {
		var event RetryEvent
		err = l.abi.UnpackIntoInterface(&event, "Retry", dl.Data)
		if err != nil {
			log.Error().Msgf(
				"failed unpacking retry event with txhash %s, because of: %+v", dl.TxHash.Hex(), err,
			)
			continue
		}
		retryEvents = append(retryEvents, event)
	}

	return retryEvents, nil
}

func (l *Listener) FetchKeygenEvents(ctx context.Context, contractAddress common.Address, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error) {
	logs, err := l.client.FetchEventLogs(ctx, contractAddress, string(StartKeygenSig), startBlock, endBlock)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func (l *Listener) FetchRefreshEvents(ctx context.Context, contractAddress common.Address, startBlock *big.Int, endBlock *big.Int) ([]*Refresh, error) {
	logs, err := l.client.FetchEventLogs(ctx, contractAddress, string(KeyRefreshSig), startBlock, endBlock)
	if err != nil {
		return nil, err
	}
	refreshEvents := make([]*Refresh, 0)

	for _, re := range logs {
		r, err := l.UnpackRefresh(l.abi, re.Data)
		if err != nil {
			log.Err(err).Msgf("failed unpacking refresh event log")
			continue
		}

		refreshEvents = append(refreshEvents, r)
	}

	return refreshEvents, nil
}

func (l *Listener) UnpackRefresh(abi abi.ABI, data []byte) (*Refresh, error) {
	var rl Refresh

	err := abi.UnpackIntoInterface(&rl, "KeyRefresh", data)
	if err != nil {
		return &Refresh{}, err
	}

	return &rl, nil
}
