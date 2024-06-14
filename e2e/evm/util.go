// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package evm

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/sygmaprotocol/sygma-core/chains/evm/client"
	"github.com/sygmaprotocol/sygma-core/chains/evm/transactor/gas"
	"github.com/sygmaprotocol/sygma-core/crypto/secp256k1"

	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/events"
	"github.com/ChainSafe/sygma-relayer/chains/evm/listener/depositHandlers"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	ethereumTypes "github.com/ethereum/go-ethereum/core/types"
)

var TestTimeout = time.Minute * 4

var (
	BasicFee        = big.NewInt(100000000000000)
	AdminAccount, _ = secp256k1.NewKeypairFromString("cc2c32b154490f09f70c1c8d4b997238448d649e0777495863db231c4ced3616")
)

const (
	ETHEndpoint1 = "ws://localhost:8545"
	ETHEndpoint2 = "ws://localhost:8547"
	OracleFee    = uint16(500) // 5% -  multiplied by 100 to not lose precision on contract side
	GasUsed      = uint32(2000000000)
)

type Client interface {
	LatestBlock() (*big.Int, error)
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- ethereumTypes.Log) (ethereum.Subscription, error)
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]ethereumTypes.Log, error)
}

type EVMClient interface {
	client.Client
	gas.GasPriceClient
	ChainID(ctx context.Context) (*big.Int, error)
}

type BridgeConfig struct {
	BridgeAddr common.Address

	Erc20Addr        common.Address
	Erc20HandlerAddr common.Address
	Erc20ResourceID  [32]byte

	Erc20LockReleaseAddr        common.Address
	Erc20LockReleaseHandlerAddr common.Address
	Erc20LockReleaseResourceID  [32]byte

	GenericHandlerAddr common.Address
	AssetStoreAddr     common.Address
	GenericResourceID  [32]byte

	PermissionlessGenericHandlerAddr common.Address
	PermissionlessGenericResourceID  [32]byte

	Erc721Addr        common.Address
	Erc721HandlerAddr common.Address
	Erc721ResourceID  [32]byte

	Erc1155Addr        common.Address
	Erc1155HandlerAddr common.Address
	Erc1155ResourceID  [32]byte

	BasicFeeHandlerAddr common.Address
	FeeRouterAddress    common.Address
	BasicFee            *big.Int

	MaxGasPrice   *big.Int
	GasMultiplier *big.Float
}

var DEFAULT_CONFIG = BridgeConfig{
	BridgeAddr: common.HexToAddress("0x6CdE2Cd82a4F8B74693Ff5e194c19CA08c2d1c68"),

	Erc20Addr:        common.HexToAddress("0x78E5b9cEC9aEA29071f070C8cC561F692B3511A6"),
	Erc20HandlerAddr: common.HexToAddress("0x02091EefF969b33A5CE8A729DaE325879bf76f90"),
	Erc20ResourceID:  SliceTo32Bytes(common.LeftPadBytes([]byte{0}, 31)),

	Erc20LockReleaseAddr:        common.HexToAddress("0x1ED1d77911944622FCcDDEad8A731fd77E94173e"),
	Erc20LockReleaseHandlerAddr: common.HexToAddress("0x02091EefF969b33A5CE8A729DaE325879bf76f90"),
	Erc20LockReleaseResourceID:  SliceTo32Bytes(common.LeftPadBytes([]byte{3}, 31)),

	Erc721Addr:        common.HexToAddress("0xa4640d1315Be1f88aC4F81546AA2C785cf247C31"),
	Erc721HandlerAddr: common.HexToAddress("0xC2D334e2f27A9dB2Ed8C4561De86C1A00EBf6760"),
	Erc721ResourceID:  SliceTo32Bytes(common.LeftPadBytes([]byte{2}, 31)),

	GenericHandlerAddr: common.HexToAddress("0xF956Ba663bd563f585e00D5973E06b443E5C4D65"),
	GenericResourceID:  SliceTo32Bytes(common.LeftPadBytes([]byte{1}, 31)),
	AssetStoreAddr:     common.HexToAddress("0xa2451c8553371E754F5e93A440aDcCa1c0DcF395"),

	PermissionlessGenericHandlerAddr: common.HexToAddress("0x156fA85e1df5d69B0F138dcEbAa5a14ca640FaED"),
	PermissionlessGenericResourceID:  SliceTo32Bytes(common.LeftPadBytes([]byte{5}, 31)),

	Erc1155Addr:        common.HexToAddress("0xFfd243c2C27e303e6d96aA4F7b3840Bf7209F0d7"),
	Erc1155HandlerAddr: common.HexToAddress("0x9Fd58882b82EFaD2867f7eaB43539907bc07C360"),
	Erc1155ResourceID:  SliceTo32Bytes(common.LeftPadBytes([]byte{4}, 31)),

	BasicFeeHandlerAddr: common.HexToAddress("0x1CcB4231f2ff299E1E049De76F0a1D2B415C563A"),
	FeeRouterAddress:    common.HexToAddress("0xF28c11CB14C6d2B806f99EA8b138F65e74a1Ed66"),
	BasicFee:            BasicFee,
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

func SliceTo32Bytes(in []byte) [32]byte {
	var res [32]byte
	copy(res[:], in)
	return res
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

func ConstructErc1155DepositData(destRecipient []byte, tokenIds *big.Int, amounts *big.Int, metadata []byte) ([]byte, error) {
	erc1155Type, err := depositHandlers.GetErc1155Type()
	if err != nil {
		return nil, err
	}

	payload := []interface{}{
		[]*big.Int{
			tokenIds,
		},
		[]*big.Int{
			amounts,
		},
		destRecipient,
		[]byte{},
	}
	data, err := erc1155Type.Pack(payload...)

	if err != nil {
		return nil, err
	}
	return data, nil
}
