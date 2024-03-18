// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package evm

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"time"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmgaspricer"
	"github.com/ChainSafe/chainbridge-core/types"

	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/events"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethereumTypes "github.com/ethereum/go-ethereum/core/types"
)

var TestTimeout = time.Minute * 4
var BasicFee = big.NewInt(100000000000000)
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

type BridgeConfig struct {
	BridgeAddr common.Address

	Erc20Addr        common.Address
	Erc20HandlerAddr common.Address
	Erc20ResourceID  types.ResourceID

	Erc20LockReleaseAddr        common.Address
	Erc20LockReleaseHandlerAddr common.Address
	Erc20LockReleaseResourceID  types.ResourceID

	GenericHandlerAddr common.Address
	AssetStoreAddr     common.Address
	GenericResourceID  types.ResourceID

	PermissionlessGenericHandlerAddr common.Address
	PermissionlessGenericResourceID  types.ResourceID

	Erc721Addr        common.Address
	Erc721HandlerAddr common.Address
	Erc721ResourceID  types.ResourceID

	Erc1155Addr        common.Address
	Erc1155HandlerAddr common.Address
	Erc1155ResourceID  types.ResourceID

	BasicFeeHandlerAddr      common.Address
	FeeRouterAddress         common.Address
	FeeHandlerWithOracleAddr common.Address
	BasicFee                 *big.Int
	OracleFee                uint16
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
	ch := make(chan ethereumTypes.Log)
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

func ConstructFeeData(feeOracleSignature string, feeDataHash string, amountToDeposit *big.Int) []byte {
	decodedFeeOracleSignature, _ := hex.DecodeString(feeOracleSignature)
	decodedFeeData, _ := hex.DecodeString(feeDataHash)
	amountToDepositBytes := calls.SliceTo32Bytes(common.LeftPadBytes(amountToDeposit.Bytes(), 32))
	feeData := append(decodedFeeData, decodedFeeOracleSignature...)
	feeData = append(feeData, amountToDepositBytes[:]...)
	return feeData
}
