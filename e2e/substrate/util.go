// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package substrate

import (
	"context"
	"encoding/binary"
	"errors"
	"math/big"
	"time"

	substrateTypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/connection"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmgaspricer"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethereumTypes "github.com/ethereum/go-ethereum/core/types"
)

var TestTimeout = time.Minute * 4
var BasicFee = big.NewInt(1000000000000000)
var OracleFee = uint16(500) // 5% -  multiplied by 100 to not lose precision on contract side
var GasUsed = uint32(2000000000)
var FeeOracleAddress = common.HexToAddress("0x70B7D7448982b15295150575541D1d3b862f7FE9")

type Client interface {
	LatestBlock() (*big.Int, error)
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- ethereumTypes.Log) (ethereum.Subscription, error)
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]ethereumTypes.Log, error)
}

type EVMClient interface {
	calls.ContractCallerDispatcher
	evmgaspricer.GasPriceClient
	ChainID(ctx context.Context) (*big.Int, error)
}

func WaitForProposalExecuted(connection *connection.Connection, beforeBalance substrateTypes.U128, key []byte) error {
	timeout := time.After(TestTimeout)
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			done, err := checkBalance(beforeBalance, connection, key)
			if err != nil {
				ticker.Stop()
				return err
			}
			if done {
				ticker.Stop()
				return nil
			}
		case <-timeout:
			ticker.Stop()
			return errors.New("test timed out waiting for proposal execution event")
		}
	}
}

func checkBalance(beforeBalance substrateTypes.U128, connection *connection.Connection, key []byte) (bool, error) {
	meta := connection.GetMetadata()
	var acc Account
	var assetId uint32 = 2000
	assetIdSerialized := make([]byte, 4)
	binary.LittleEndian.PutUint32(assetIdSerialized, assetId)

	key, _ = substrateTypes.CreateStorageKey(&meta, "Assets", "Account", assetIdSerialized, key)
	_, err := connection.RPC.State.GetStorageLatest(key, &acc)
	if err != nil {
		return false, err
	}
	destBalanceAfter := acc.Balance
	if destBalanceAfter.Int.Cmp(beforeBalance.Int) == 1 {
		return true, nil
	} else {
		return false, nil
	}
}

type Account struct {
	Balance substrateTypes.U128
}
