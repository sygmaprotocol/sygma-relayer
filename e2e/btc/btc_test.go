// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package btc_test

import (
	"context"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/core/types"
	evmClient "github.com/sygmaprotocol/sygma-core/chains/evm/client"
	"github.com/sygmaprotocol/sygma-core/chains/evm/transactor"
	"github.com/sygmaprotocol/sygma-core/chains/evm/transactor/gas"
	"github.com/sygmaprotocol/sygma-core/chains/evm/transactor/signAndSend"
	"github.com/sygmaprotocol/sygma-core/chains/evm/transactor/transaction"

	"math/big"
	"testing"

	"github.com/ChainSafe/sygma-relayer/chains/btc/connection"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/bridge"

	"github.com/ChainSafe/sygma-relayer/e2e/btc"
	"github.com/ChainSafe/sygma-relayer/e2e/evm"
	"github.com/ChainSafe/sygma-relayer/e2e/evm/contracts/erc20"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

var amountToDeposit = big.NewInt(100000000)

type TestClient interface {
	evm.EVMClient
	LatestBlock() (*big.Int, error)
	CodeAt(ctx context.Context, contractAddress common.Address, block *big.Int) ([]byte, error)
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]types.Log, error)
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
}

func Test_EVMBtc(t *testing.T) {
	evmConfig := evm.DEFAULT_CONFIG
	evmConfig.Erc20LockReleaseAddr = common.HexToAddress("0x8f5b7716a0A5f94Ea10590F9070442f285a31116")
	evmConfig.Erc20LockReleaseResourceID = evm.SliceTo32Bytes(common.LeftPadBytes([]byte{9}, 31))

	ethClient, err := evmClient.NewEVMClient(evm.ETHEndpoint1, evm.AdminAccount)
	if err != nil {
		panic(err)
	}
	gasPricer := gas.NewStaticGasPriceDeterminant(ethClient, nil)

	suite.Run(
		t,
		NewEVMBtcTestSuite(
			transaction.NewTransaction,
			ethClient,
			gasPricer,
			evmConfig,
		),
	)
}

func NewEVMBtcTestSuite(
	fabric transaction.TxFabric,
	evmClient TestClient,
	gasPricer signAndSend.GasPricer,
	evmConfig evm.BridgeConfig,
) *IntegrationTestSuite {
	return &IntegrationTestSuite{
		fabric:    fabric,
		evmClient: evmClient,
		gasPricer: gasPricer,
		evmConfig: evmConfig,
	}
}

type IntegrationTestSuite struct {
	suite.Suite
	fabric    transaction.TxFabric
	evmClient TestClient
	gasPricer signAndSend.GasPricer
	evmConfig evm.BridgeConfig
}

func (s *IntegrationTestSuite) SetupSuite() {
	// EVM side preparation
	evmTransactor := signAndSend.NewSignAndSendTransactor(s.fabric, s.gasPricer, s.evmClient)
	mintTo := s.evmClient.From()
	amountToMint := big.NewInt(0).Mul(big.NewInt(5000000000000000), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))
	amountToApprove := big.NewInt(0).Mul(big.NewInt(100000), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))
	erc20LRContract := erc20.NewERC20Contract(s.evmClient, s.evmConfig.Erc20LockReleaseAddr, evmTransactor)
	_, err := erc20LRContract.MintTokens(mintTo, amountToMint, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}
	_, err = erc20LRContract.ApproveTokens(s.evmConfig.Erc20LockReleaseHandlerAddr, amountToApprove, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}
}

func (s *IntegrationTestSuite) Test_Erc20Deposit_EVM_to_Substrate() {
	conn, err := connection.NewBtcConnection(btc.BtcEndpoint)
	s.Nil(err)

	addr, err := btcutil.DecodeAddress("bcrt1pja8aknn7te4empmghnyqnrtjqn0lyg5zy3p5jsdp4le930wnpnxsrtd3ht", &chaincfg.RegressionNetParams)
	s.Nil(err)
	balanceBefore, err := conn.ListUnspentMinMaxAddresses(0, 1000, []btcutil.Address{addr})
	s.Nil(err)
	s.Equal(len(balanceBefore), 0)

	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric, s.gasPricer, s.evmClient)
	bridgeContract1 := bridge.NewBridgeContract(s.evmClient, s.evmConfig.BridgeAddr, transactor1)
	erc20DepositData := evm.ConstructErc20DepositData([]byte("bcrt1pja8aknn7te4empmghnyqnrtjqn0lyg5zy3p5jsdp4le930wnpnxsrtd3ht"), amountToDeposit)
	_, err = bridgeContract1.ExecuteTransaction("deposit", transactor.TransactOptions{Value: s.evmConfig.BasicFee}, uint8(4), s.evmConfig.Erc20LockReleaseResourceID, erc20DepositData, []byte{})
	s.Nil(err)

	err = btc.WaitForProposalExecuted(conn, addr)
	s.Nil(err)

	balanceAfter, err := conn.ListUnspentMinMaxAddresses(0, 1000, []btcutil.Address{addr})
	s.Nil(err)
	s.Equal(balanceAfter[0].Amount*100000000, float64(amountToDeposit.Int64()))

}
