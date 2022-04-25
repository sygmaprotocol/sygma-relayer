package events

import (
	"context"
	"math/big"
	"strings"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/consts"
	"github.com/ChainSafe/chainbridge-core/util"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

type ChainClient interface {
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error)
}

type Listener struct {
	client ChainClient
}

func NewListener(client ChainClient) *Listener {
	return &Listener{
		client: client,
	}
}

func (l *Listener) FetchDeposits(ctx context.Context, contractAddress common.Address, startBlock *big.Int, endBlock *big.Int) ([]*Deposit, error) {
	logs, err := l.client.FetchEventLogs(ctx, contractAddress, string(util.Deposit), startBlock, endBlock)
	if err != nil {
		return nil, err
	}
	deposits := make([]*Deposit, 0)

	abi, err := abi.JSON(strings.NewReader(consts.BridgeABI))
	if err != nil {
		return nil, err
	}

	for _, dl := range logs {
		d, err := l.UnpackDeposit(abi, dl.Data)
		if err != nil {
			log.Error().Msgf("failed unpacking deposit event log: %v", err)
			continue
		}

		d.SenderAddress = common.BytesToAddress(dl.Topics[1].Bytes())
		log.Debug().Msgf("Found deposit log in block: %d, TxHash: %s, contractAddress: %s, sender: %s", dl.BlockNumber, dl.TxHash, dl.Address, d.SenderAddress)

		deposits = append(deposits, d)
	}

	return deposits, nil
}

func (l *Listener) UnpackDeposit(abi abi.ABI, data []byte) (*Deposit, error) {
	var dl Deposit

	err := abi.UnpackIntoInterface(&dl, "Deposit", data)
	if err != nil {
		return &Deposit{}, err
	}

	return &dl, nil
}
