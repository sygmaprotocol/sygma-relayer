// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package evm_test

import (
	"context"
	"math/big"
	"math/rand"
	"sync"
	"testing"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/centrifuge"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/erc721"
	substrateTypes "github.com/centrifuge/go-substrate-rpc-client/types"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/erc20"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/signAndSend"
	"github.com/ChainSafe/chainbridge-core/e2e/dummy"
	"github.com/ChainSafe/chainbridge-core/keystore"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/deployutils"
	"github.com/ChainSafe/sygma-relayer/e2e/evm"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

type TestClient interface {
	deployutils.EVMClient
	LatestBlock() (*big.Int, error)
	CodeAt(ctx context.Context, contractAddress common.Address, block *big.Int) ([]byte, error)
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]types.Log, error)
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
}

const ETHEndpoint1 = "ws://localhost:8546"
const ETHEndpoint2 = "ws://localhost:8548"

// Alice key is used by the relayer, Charlie key is used as admin and depositer
func Test_EVM2EVM(t *testing.T) {
	config := deployutils.BridgeConfig{
		BridgeAddr: common.HexToAddress("0xF75ABb9ABED5975d1430ddCF420bEF954C8F5235"),

		Erc20Addr:        common.HexToAddress("0xDA8556C2485048eee3dE91085347c3210785323c"),
		Erc20HandlerAddr: common.HexToAddress("0x7ec51Af51bf6f6f4e3C2E87096381B2cf94f6d74"),
		Erc20ResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{0}, 31)),

		Erc20LockReleaseAddr:        common.HexToAddress("0xbD259407A231Ad2a50df1e8CBaCe9A5E63EB65D5"),
		Erc20LockReleaseHandlerAddr: common.HexToAddress("0x7ec51Af51bf6f6f4e3C2E87096381B2cf94f6d74"),
		Erc20LockReleaseResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{3}, 31)),

		Erc721Addr:        common.HexToAddress("0xd6D787253cc022E6839583aD0cBECfc9c60b581c"),
		Erc721HandlerAddr: common.HexToAddress("0x1cd88Fa5848389E4027d29B267BAB561300CEA2A"),
		Erc721ResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{2}, 31)),

		GenericHandlerAddr: common.HexToAddress("0xf1a8fDee59ecc8bDbAAA7cC0757876177d0FB255"),
		GenericResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{1}, 31)),
		AssetStoreAddr:     common.HexToAddress("0x1C9D948eddE23f66f8c816241C7587bC2845fA7d"),

		BasicFeeHandlerAddr:      common.HexToAddress("0x7ba8A49750Ea49783a456A71096923723E6566ee"),
		FeeHandlerWithOracleAddr: common.HexToAddress("0xA81cC6305C6f62Ccd81fc7D1E2EC6F804aCB4512"),
		FeeRouterAddress:         common.HexToAddress("0xA8254f6184b82D7307257966b95D7569BD751a90"),
		BasicFee:                 deployutils.BasicFee,
		OracleFee:                deployutils.OracleFee,
	}

	ethClient1, err := evmclient.NewEVMClient(ETHEndpoint1, deployutils.CharlieKp.PrivateKey())
	if err != nil {
		panic(err)
	}
	gasPricer1 := dummy.NewStaticGasPriceDeterminant(ethClient1, nil)

	ethClient2, err := evmclient.NewEVMClient(ETHEndpoint2, deployutils.CharlieKp.PrivateKey())
	if err != nil {
		panic(err)
	}
	gasPricer2 := dummy.NewStaticGasPriceDeterminant(ethClient2, nil)

	suite.Run(
		t,
		NewEVM2EVMTestSuite(
			evmtransaction.NewTransaction,
			evmtransaction.NewTransaction,
			ethClient1,
			ethClient2,
			gasPricer1,
			gasPricer2,
			config,
			config,
		),
	)
}

func NewEVM2EVMTestSuite(
	fabric1, fabric2 calls.TxFabric,
	client1, client2 TestClient,
	gasPricer1, gasPricer2 calls.GasPricer,
	config1, config2 deployutils.BridgeConfig,
) *IntegrationTestSuite {
	return &IntegrationTestSuite{
		fabric1:    fabric1,
		fabric2:    fabric2,
		client1:    client1,
		client2:    client2,
		gasPricer1: gasPricer1,
		gasPricer2: gasPricer2,
		config1:    config1,
		config2:    config2,
	}
}

type IntegrationTestSuite struct {
	suite.Suite
	client1    TestClient
	client2    TestClient
	gasPricer1 calls.GasPricer
	gasPricer2 calls.GasPricer
	fabric1    calls.TxFabric
	fabric2    calls.TxFabric
	config1    deployutils.BridgeConfig
	config2    deployutils.BridgeConfig
}

// SetupSuite waits until all contracts are deployed
func (s *IntegrationTestSuite) SetupSuite() {
	log.Info().Msg("Waiting for Bridge to set")
	err := evm.WaitUntilBridgeReady(s.client2, s.config2.BasicFeeHandlerAddr)
	if err != nil {
		panic(err)
	}
	log.Info().Msg("Bridge is set")
}

var amountToDeposit = big.NewInt(100)
var oracleFeeInWei, _ = new(big.Int).SetString("63795456000000000000", 0)
var amountToMint, _ = new(big.Int).SetString("5000000000000000000000", 0)
var amountToApprove, _ = new(big.Int).SetString("10000000000000000000000", 0)

func (s *IntegrationTestSuite) Test_Erc20Deposit() {
	dstAddr := keystore.TestKeyRing.EthereumKeys[keystore.BobKey].CommonAddress()

	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	erc20Contract1 := erc20.NewERC20Contract(s.client1, s.config1.Erc20Addr, transactor1)
	bridgeContract1 := bridge.NewBridgeContract(s.client1, s.config1.BridgeAddr, transactor1)

	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)
	erc20Contract2 := erc20.NewERC20Contract(s.client2, s.config2.Erc20Addr, transactor2)

	_, err := erc20Contract1.MintTokens(s.config1.Erc20HandlerAddr, amountToMint, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	if err != nil {
		return
	}

	_, err = erc20Contract1.MintTokens(s.client1.From(), amountToMint, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	if err != nil {
		return
	}

	_, err = erc20Contract1.ApproveTokens(s.config1.Erc20HandlerAddr, amountToApprove, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	s.Nil(err)

	_, err = erc20Contract1.ApproveTokens(s.config1.FeeHandlerWithOracleAddr, amountToApprove, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	s.Nil(err)

	senderBalBefore, err := erc20Contract1.GetBalance(deployutils.CharlieKp.CommonAddress())
	s.Nil(err)
	destBalanceBefore, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)
	handlerBalanceBefore, err := erc20Contract1.GetBalance(s.config1.FeeHandlerWithOracleAddr)
	s.Nil(err)

	var feeOracleSignature = "2ecbfbc1db3c1976987a32c5cc3043a5fbe2468c86472ab0ac8ea9a3b97291e3585a655596580ad76a0c06eeb1ce71d75d6799dc34dce7cfeea3048351000acb1b"
	var feeDataHash = "000000000000000000000000000000000000000000000000000194b9a2ecd000000000000000000000000000000000000000000000000000dd55bf4eab0400000000000000000000000000000000000000000000000000000000000077359400000000000000000000000000000000000000000000000000000000006918d61d000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000"
	var feeData = evm.ConstructFeeData(feeOracleSignature, feeDataHash, amountToDeposit)

	depositTxHash, err := bridgeContract1.Erc20Deposit(dstAddr, amountToDeposit, s.config1.Erc20ResourceID, 2, feeData,
		transactor.TransactOptions{
			Priority: uint8(2), // fast
		})
	s.Nil(err)

	log.Debug().Msgf("deposit hash %s", depositTxHash.Hex())

	depositTx, _, err := s.client1.TransactionByHash(context.Background(), *depositTxHash)
	s.Nil(err)
	// check gas price of deposit tx - 140 gwei
	s.Equal(big.NewInt(140000000000), depositTx.GasPrice())

	err = evm.WaitForProposalExecuted(s.client2, s.config2.BridgeAddr)
	s.Nil(err)

	senderBalAfter, err := erc20Contract1.GetBalance(s.client1.From())
	s.Nil(err)
	s.Equal(-1, senderBalAfter.Cmp(senderBalBefore))

	destBalanceAfter, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)
	//Balance has increased
	s.Equal(1, destBalanceAfter.Cmp(destBalanceBefore))

	// Check that FeeHandler token balance increased
	handlerBalanceAfter, _ := erc20Contract1.GetBalance(s.config1.FeeHandlerWithOracleAddr)
	s.Nil(err)
	s.Equal(handlerBalanceAfter, handlerBalanceBefore.Add(handlerBalanceBefore, oracleFeeInWei))
}

func (s *IntegrationTestSuite) Test_Erc721Deposit() {
	tokenId := big.NewInt(int64(rand.Intn(1000)))
	metadata := "metadata.url"

	txOptions := transactor.TransactOptions{
		Priority: uint8(2), // fast
	}

	dstAddr := keystore.TestKeyRing.EthereumKeys[keystore.BobKey].CommonAddress()

	// erc721 contract for evm1
	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	erc721Contract1 := erc721.NewErc721Contract(s.client1, s.config1.Erc721Addr, transactor1)
	bridgeContract1 := bridge.NewBridgeContract(s.client1, s.config1.BridgeAddr, transactor1)

	// erc721 contract for evm2
	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)
	erc721Contract2 := erc721.NewErc721Contract(s.client2, s.config2.Erc721Addr, transactor2)

	// Mint token and give approval
	// This is done here so token only exists on evm1
	_, err := erc721Contract1.Mint(tokenId, metadata, s.client1.From(), txOptions)
	s.Nil(err, "Mint failed")
	_, err = erc721Contract1.Approve(tokenId, s.config1.Erc721HandlerAddr, txOptions)
	s.Nil(err, "Approve failed")

	// Check on evm1 if initial owner is admin
	initialOwner, err := erc721Contract1.Owner(tokenId)
	s.Nil(err)
	s.Equal(initialOwner.String(), s.client1.From().String())

	// Check on evm2 token doesn't exist
	_, err = erc721Contract2.Owner(tokenId)
	s.Error(err)

	handlerBalanceBefore, err := s.client1.BalanceAt(context.TODO(), s.config1.BasicFeeHandlerAddr, nil)
	s.Nil(err)

	depositTxHash, err := bridgeContract1.Erc721Deposit(
		tokenId, metadata, dstAddr, s.config1.Erc721ResourceID, 2, nil, transactor.TransactOptions{
			Value: s.config1.BasicFee,
		},
	)
	s.Nil(err)

	depositTx, _, err := s.client1.TransactionByHash(context.Background(), *depositTxHash)
	s.Nil(err)
	// check gas price of deposit tx - 50 gwei (slow)
	s.Equal(big.NewInt(50000000000), depositTx.GasPrice())

	err = evm.WaitForProposalExecuted(s.client2, s.config2.BridgeAddr)
	s.Nil(err)

	// Check on evm1 that token is burned
	_, err = erc721Contract1.Owner(tokenId)
	s.Error(err)

	// Check on evm2 that token is minted to destination address
	owner, err := erc721Contract2.Owner(tokenId)
	s.Nil(err)
	s.Equal(dstAddr.String(), owner.String())

	// Check that FeeHandler ETH balance increased
	handlerBalanceAfter, err := s.client1.BalanceAt(context.TODO(), s.config1.BasicFeeHandlerAddr, nil)
	s.Nil(err)
	s.Equal(handlerBalanceAfter, big.NewInt(0).Add(handlerBalanceBefore, s.config1.BasicFee))
}

func (s *IntegrationTestSuite) Test_GenericDeposit() {
	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)

	bridgeContract1 := bridge.NewBridgeContract(s.client1, s.config1.BridgeAddr, transactor1)
	assetStoreContract2 := centrifuge.NewAssetStoreContract(s.client2, s.config2.AssetStoreAddr, transactor2)

	hash, _ := substrateTypes.GetHash(substrateTypes.NewI64(int64(rand.Int())))

	handlerBalanceBefore, err := s.client1.BalanceAt(context.TODO(), s.config1.BasicFeeHandlerAddr, nil)
	s.Nil(err)

	depositTxHash, err := bridgeContract1.GenericDeposit(hash[:], s.config1.GenericResourceID, 2, nil, transactor.TransactOptions{
		Value: s.config1.BasicFee,
	})
	s.Nil(err)

	depositTx, _, err := s.client1.TransactionByHash(context.Background(), *depositTxHash)
	s.Nil(err)
	// check gas price of deposit tx - 140 gwei
	s.Equal(big.NewInt(50000000000), depositTx.GasPrice())

	err = evm.WaitForProposalExecuted(s.client2, s.config2.BridgeAddr)
	s.Nil(err)
	// Asset hash sent is stored in centrifuge asset store contract
	exists, err := assetStoreContract2.IsCentrifugeAssetStored(hash)
	s.Nil(err)
	s.Equal(true, exists)

	handlerBalanceAfter, err := s.client1.BalanceAt(context.TODO(), s.config1.BasicFeeHandlerAddr, nil)
	s.Nil(err)
	s.Equal(handlerBalanceAfter, big.NewInt(0).Add(handlerBalanceBefore, s.config1.BasicFee))
}

func (s *IntegrationTestSuite) Test_RetryDeposit() {
	dstAddr := keystore.TestKeyRing.EthereumKeys[keystore.BobKey].CommonAddress()

	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	erc20Contract1 := erc20.NewERC20Contract(s.client1, s.config1.Erc20LockReleaseAddr, transactor1)
	bridgeContract1 := bridge.NewBridgeContract(s.client1, s.config1.BridgeAddr, transactor1)

	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)
	erc20Contract2 := erc20.NewERC20Contract(s.client2, s.config2.Erc20LockReleaseAddr, transactor2)

	_, err := erc20Contract1.MintTokens(s.client1.From(), amountToMint, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	if err != nil {
		return
	}

	_, err = erc20Contract1.ApproveTokens(s.config1.Erc20HandlerAddr, amountToApprove, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	s.Nil(err)

	_, err = erc20Contract1.ApproveTokens(s.config1.FeeHandlerWithOracleAddr, amountToApprove, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	s.Nil(err)

	senderBalBefore, err := erc20Contract1.GetBalance(deployutils.CharlieKp.CommonAddress())
	s.Nil(err)
	destBalanceBefore, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)
	handlerBalanceBefore, err := erc20Contract1.GetBalance(s.config1.FeeHandlerWithOracleAddr)
	s.Nil(err)

	var feeOracleSignature = "ba825f046f4a40bbe99d50e16a93afa99c524ad90cb64c0523d7fa79adb03b2705fd53a4af0f94ae6b2ea8da7f2b6497c41774e9f93a116cf40714553b53db511c"
	var feeDataHash = "000000000000000000000000000000000000000000000000000194b9a2ecd000000000000000000000000000000000000000000000000000dd55bf4eab04000000000000000000000000000000000000000000000000000000000000773594000000000000000000000000000000000000000000000000000000000069191ba6000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000300"
	var feeData = evm.ConstructFeeData(feeOracleSignature, feeDataHash, amountToDeposit)

	depositTxHash, err := bridgeContract1.Erc20Deposit(dstAddr, amountToDeposit, s.config1.Erc20LockReleaseResourceID, 2, feeData,
		transactor.TransactOptions{
			Priority: uint8(2), // fast
		})
	s.Nil(err)

	log.Debug().Msgf("deposit hash %s", depositTxHash.Hex())

	depositTx, _, err := s.client1.TransactionByHash(context.Background(), *depositTxHash)
	s.Nil(err)

	// check gas price of deposit tx - 140 gwei
	s.Equal(big.NewInt(140000000000), depositTx.GasPrice())

	err = evm.WaitForProposalExecuted(s.client2, s.config2.BridgeAddr)
	s.NotNil(err)

	_, err = erc20Contract2.MintTokens(s.config2.Erc20HandlerAddr, amountToMint, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	if err != nil {
		return
	}

	retryTxHash, err := bridgeContract1.Retry(*depositTxHash, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	if err != nil {
		return
	}
	s.Nil(err)
	s.NotNil(retryTxHash)

	err = evm.WaitForProposalExecuted(s.client2, s.config2.BridgeAddr)
	s.Nil(err)

	senderBalAfter, err := erc20Contract1.GetBalance(s.client1.From())
	s.Nil(err)
	s.Equal(-1, senderBalAfter.Cmp(senderBalBefore))

	destBalanceAfter, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)
	//Balance has increased
	s.Equal(1, destBalanceAfter.Cmp(destBalanceBefore))

	// Check that FeeHandler token balance increased
	handlerBalanceAfter, _ := erc20Contract1.GetBalance(s.config1.FeeHandlerWithOracleAddr)
	s.Nil(err)
	s.Equal(handlerBalanceAfter, handlerBalanceBefore.Add(handlerBalanceBefore, oracleFeeInWei))
}

func (s *IntegrationTestSuite) Test_MultipleDeposits() {
	dstAddr := keystore.TestKeyRing.EthereumKeys[keystore.BobKey].CommonAddress()

	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	erc20Contract1 := erc20.NewERC20Contract(s.client1, s.config1.Erc20Addr, transactor1)
	bridgeContract1 := bridge.NewBridgeContract(s.client1, s.config1.BridgeAddr, transactor1)

	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)
	erc20Contract2 := erc20.NewERC20Contract(s.client2, s.config2.Erc20Addr, transactor2)

	_, err := erc20Contract1.MintTokens(s.config1.Erc20HandlerAddr, amountToMint, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	if err != nil {
		return
	}

	_, err = erc20Contract1.MintTokens(s.client1.From(), amountToMint, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	if err != nil {
		return
	}

	_, err = erc20Contract1.ApproveTokens(s.config1.Erc20HandlerAddr, amountToApprove, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	s.Nil(err)

	_, err = erc20Contract1.ApproveTokens(s.config1.FeeHandlerWithOracleAddr, amountToApprove, transactor.TransactOptions{
		Priority: uint8(2), // fast
	})
	s.Nil(err)

	senderBalBefore, err := erc20Contract1.GetBalance(deployutils.CharlieKp.CommonAddress())
	s.Nil(err)
	destBalanceBefore, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)
	handlerBalanceBefore, err := erc20Contract1.GetBalance(s.config1.FeeHandlerWithOracleAddr)
	s.Nil(err)

	var feeOracleSignature = "2ecbfbc1db3c1976987a32c5cc3043a5fbe2468c86472ab0ac8ea9a3b97291e3585a655596580ad76a0c06eeb1ce71d75d6799dc34dce7cfeea3048351000acb1b"
	var feeDataHash = "000000000000000000000000000000000000000000000000000194b9a2ecd000000000000000000000000000000000000000000000000000dd55bf4eab0400000000000000000000000000000000000000000000000000000000000077359400000000000000000000000000000000000000000000000000000000006918d61d000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000"
	var feeData = evm.ConstructFeeData(feeOracleSignature, feeDataHash, amountToDeposit)

	numOfDeposits := 25
	totalDepositAmount := big.NewInt(0).Mul(amountToDeposit, big.NewInt(int64(numOfDeposits)))

	var wg sync.WaitGroup
	for i := 0; i < numOfDeposits; i++ {
		go func() {
			_, err := bridgeContract1.Erc20Deposit(dstAddr, amountToDeposit, s.config1.Erc20ResourceID, 2, feeData,
				transactor.TransactOptions{
					Priority: uint8(2), // fast
				})
			wg.Add(1)
			defer wg.Done()
			s.Nil(err)
		}()
	}
	wg.Wait()
	err = evm.WaitForProposalExecuted(s.client2, s.config2.BridgeAddr)
	s.Nil(err)

	totalFees := big.NewInt(0).Mul(oracleFeeInWei, big.NewInt(int64(numOfDeposits)))

	destBalanceAfter, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)
	//Balance has increased
	s.Equal(1, destBalanceAfter.Cmp(destBalanceBefore))

	senderBalAfter, err := erc20Contract1.GetBalance(s.client1.From())
	s.Nil(err)
	senderBalanceDifference := big.NewInt(0).Sub(senderBalBefore, senderBalAfter)
	s.Equal(totalDepositAmount, big.NewInt(0).Sub(senderBalanceDifference, totalFees))

	// Check that FeeHandler token balance increased
	handlerBalanceAfter, _ := erc20Contract1.GetBalance(s.config1.FeeHandlerWithOracleAddr)
	s.Nil(err)
	s.Equal(handlerBalanceAfter, big.NewInt(0).Add(handlerBalanceBefore, totalFees))
}
