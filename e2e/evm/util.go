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
	"github.com/ethereum/go-ethereum/common/math"
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

func ConstructPermissionlessGenericDepositData(metadata []byte, executionFunctionSig []byte, executeContractAddress []byte, metadataDepositor []byte, maxFee *big.Int) []byte {
	var data []byte
	data = append(data, common.LeftPadBytes(maxFee.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(big.NewInt(int64(len(executionFunctionSig))).Bytes(), 2)...)
	data = append(data, executionFunctionSig...)
	data = append(data, byte(len(executeContractAddress)))
	data = append(data, executeContractAddress...)
	data = append(data, byte(len(metadataDepositor)))
	data = append(data, metadataDepositor...)
	data = append(data, metadata...)
	return data
}

func constructMainDepositData(tokenStats *big.Int, destRecipient []byte) []byte {
	var data []byte
	data = append(data, math.PaddedBigBytes(tokenStats, 32)...)                            // Amount (ERC20) or Token Id (ERC721)
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(len(destRecipient))), 32)...) // length of recipient
	data = append(data, destRecipient...)                                                  // Recipient
	return data
}

func ConstructErc20DepositData(destRecipient []byte, amount *big.Int) []byte {
	data := constructMainDepositData(amount, destRecipient)
	return data
}

func ConstructErc721DepositData(destRecipient []byte, tokenId *big.Int, metadata []byte) []byte {
	data := constructMainDepositData(tokenId, destRecipient)
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(len(metadata))), 32)...) // Length of metadata
	data = append(data, metadata...)                                                  // Metadata
	return data
}

func ConstructGenericDepositData(metadata []byte) []byte {
	var data []byte
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(len(metadata))), 32)...) // Length of metadata
	data = append(data, metadata...)                                                  // Metadata
	return data
}
