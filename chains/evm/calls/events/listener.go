package events

import (
	"context"
	"fmt"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/rs/zerolog/log"
	"math/big"
	"strings"

	"github.com/ChainSafe/chainbridge-hub/chains/evm/calls/consts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

type ChainClient interface {
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error)
	WaitAndReturnTxReceipt(h common.Hash) (*ethTypes.Receipt, error)
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

func (l *Listener) FetchDepositEvent(event RetryEvent) (events.Deposit, error) {
	retryDepositTxHash := common.HexToHash(event.TxHash)
	receipt, err := l.client.WaitAndReturnTxReceipt(retryDepositTxHash)
	if err != nil {
		return events.Deposit{}, fmt.Errorf(
			"unable to fetch logs for retried deposit %s, because of: %+v", retryDepositTxHash.Hex(), err,
		)
	}

	var depositEvent events.Deposit
	for _, lg := range receipt.Logs {
		err := l.abi.UnpackIntoInterface(&depositEvent, "Deposit", lg.Data)
		if err == nil {
			break
		}
	}
	return depositEvent, nil
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

func (l *Listener) FetchRefreshEvents(ctx context.Context, contractAddress common.Address, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error) {
	logs, err := l.client.FetchEventLogs(ctx, contractAddress, string(KeyRefreshSig), startBlock, endBlock)
	if err != nil {
		return nil, err
	}

	return logs, nil
}
