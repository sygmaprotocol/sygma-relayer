package events

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ChainSafe/chainbridge-hub/chains/evm/calls/consts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

type ChainClient interface {
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error)
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

func (l *Listener) FetchRetryEvents(ctx context.Context, contractAddress common.Address, startBlock *big.Int, endBlock *big.Int) ([]RetryEvent, error) {
	logs, err := l.client.FetchEventLogs(ctx, contractAddress, string(RetrySig), startBlock, endBlock)
	if err != nil {
		return nil, err
	}

	var retryEvents []RetryEvent
	for _, log := range logs {
		var event RetryEvent
		err = l.abi.UnpackIntoInterface(event, "Retry", log.Data)
		if err != nil {
			return nil, fmt.Errorf(
				"unable to unpack retry event with txhash %s, because of: %+v", log.TxHash.Hex(), err,
			)
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
