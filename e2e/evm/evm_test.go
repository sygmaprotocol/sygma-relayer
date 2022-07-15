package evm_test

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ChainSafe/sygma-core/chains/evm/calls/contracts/centrifuge"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/contracts/erc721"
	substrateTypes "github.com/centrifuge/go-substrate-rpc-client/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/suite"

	"github.com/ChainSafe/sygma-core/chains/evm/calls"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/contracts/erc20"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/transactor/signAndSend"
	"github.com/ChainSafe/sygma-core/e2e/dummy"
	"github.com/ChainSafe/sygma-core/keystore"

	"github.com/ChainSafe/sygma/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/sygma/chains/evm/cli/local"
	"github.com/ChainSafe/sygma/e2e/evm"
)

type TestClient interface {
	local.EVMClient
	LatestBlock() (*big.Int, error)
	CodeAt(ctx context.Context, contractAddress common.Address, block *big.Int) ([]byte, error)
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]types.Log, error)
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error)
}

const ETHEndpoint1 = "ws://localhost:8546"
const ETHEndpoint2 = "ws://localhost:8548"

// Alice key is used by the relayer, Charlie key is used as admin and depositter
func Test_EVM2EVM(t *testing.T) {
	config := local.BridgeConfig{
		BridgeAddr: common.HexToAddress("0xF75ABb9ABED5975d1430ddCF420bEF954C8F5235"),

		Erc20Addr:        common.HexToAddress("0xDA8556C2485048eee3dE91085347c3210785323c"),
		Erc20HandlerAddr: common.HexToAddress("0x7ec51Af51bf6f6f4e3C2E87096381B2cf94f6d74"),
		Erc20ResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{0}, 31)),

		Erc20LockReleaseAddr:        common.HexToAddress("0xA8254f6184b82D7307257966b95D7569BD751a90"),
		Erc20LockReleaseHandlerAddr: common.HexToAddress("0x7ec51Af51bf6f6f4e3C2E87096381B2cf94f6d74"),
		Erc20LockReleaseResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{3}, 31)),

		Erc721Addr:        common.HexToAddress("0xd6D787253cc022E6839583aD0cBECfc9c60b581c"),
		Erc721HandlerAddr: common.HexToAddress("0x1cd88Fa5848389E4027d29B267BAB561300CEA2A"),
		Erc721ResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{2}, 31)),

		GenericHandlerAddr: common.HexToAddress("0xf1a8fDee59ecc8bDbAAA7cC0757876177d0FB255"),
		GenericResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{1}, 31)),
		AssetStoreAddr:     common.HexToAddress("0x1C9D948eddE23f66f8c816241C7587bC2845fA7d"),

		IsBasicFeeHandler: true,
		Fee:               big.NewInt(100000000000),
		FeeHandlerAddr:    common.HexToAddress("0xbD259407A231Ad2a50df1e8CBaCe9A5E63EB65D5"),
	}

	ethClient1, err := evmclient.NewEVMClient(ETHEndpoint1, local.CharlieKp.PrivateKey())
	if err != nil {
		panic(err)
	}
	gasPricer1 := dummy.NewStaticGasPriceDeterminant(ethClient1, nil)

	ethClient2, err := evmclient.NewEVMClient(ETHEndpoint2, local.CharlieKp.PrivateKey())
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
	config1, config2 local.BridgeConfig,
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
	config1    local.BridgeConfig
	config2    local.BridgeConfig
}

// SetupSuite waits until all contracts are deployed
func (s *IntegrationTestSuite) SetupSuite() {
	err := evm.WaitUntilBridgeReady(s.client2, s.config2.FeeHandlerAddr)
	if err != nil {
		panic(err)
	}
}

func (s *IntegrationTestSuite) Test_Erc20Deposit() {
	dstAddr := keystore.TestKeyRing.EthereumKeys[keystore.BobKey].CommonAddress()

	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	erc20Contract1 := erc20.NewERC20Contract(s.client1, s.config1.Erc20Addr, transactor1)
	bridgeContract1 := bridge.NewBridgeContract(s.client1, s.config1.BridgeAddr, transactor1)

	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)
	erc20Contract2 := erc20.NewERC20Contract(s.client2, s.config2.Erc20Addr, transactor2)

	senderBalBefore, err := erc20Contract1.GetBalance(local.CharlieKp.CommonAddress())
	s.Nil(err)
	destBalanceBefore, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)

	amountToDeposit := big.NewInt(1000000)

	depositTxHash, err := bridgeContract1.Erc20Deposit(dstAddr, amountToDeposit, s.config1.Erc20ResourceID, 2, nil,
		transactor.TransactOptions{
			Priority: uint8(2), // fast
			Value:    s.config1.Fee,
		})
	s.Nil(err)

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
}

func (s *IntegrationTestSuite) Test_Erc721Deposit() {
	tokenId := big.NewInt(int64(rand.Int()))
	metadata := "metadata.url"

	dstAddr := keystore.TestKeyRing.EthereumKeys[keystore.BobKey].CommonAddress()

	txOptions := transactor.TransactOptions{
		Priority: uint8(2), // fast
	}

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

	depositTxHash, err := bridgeContract1.Erc721Deposit(
		tokenId, metadata, dstAddr, s.config1.Erc721ResourceID, 2, nil, transactor.TransactOptions{
			Value: s.config1.Fee,
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
}

func (s *IntegrationTestSuite) Test_GenericDeposit() {
	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)

	bridgeContract1 := bridge.NewBridgeContract(s.client1, s.config1.BridgeAddr, transactor1)
	assetStoreContract2 := centrifuge.NewAssetStoreContract(s.client2, s.config2.AssetStoreAddr, transactor2)

	hash, _ := substrateTypes.GetHash(substrateTypes.NewI64(int64(rand.Int())))

	depositTxHash, err := bridgeContract1.GenericDeposit(hash[:], s.config1.GenericResourceID, 2, nil, transactor.TransactOptions{
		Priority: uint8(0), // slow
		Value:    s.config1.Fee,
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
}

func (s *IntegrationTestSuite) Test_RetryDeposit() {
	dstAddr := keystore.TestKeyRing.EthereumKeys[keystore.BobKey].CommonAddress()

	txOptions := transactor.TransactOptions{
		Priority: uint8(2), // fast
	}

	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	erc20Contract1 := erc20.NewERC20Contract(s.client1, s.config1.Erc20LockReleaseAddr, transactor1)
	bridgeContract1 := bridge.NewBridgeContract(s.client1, s.config1.BridgeAddr, transactor1)

	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)
	erc20Contract2 := erc20.NewERC20Contract(s.client2, s.config2.Erc20LockReleaseAddr, transactor2)

	senderBalBefore, err := erc20Contract1.GetBalance(local.CharlieKp.CommonAddress())
	s.Nil(err)
	destBalanceBefore, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)

	amountToDeposit := big.NewInt(1000000)

	depositTxHash, err := bridgeContract1.Erc20Deposit(dstAddr, amountToDeposit, s.config1.Erc20LockReleaseResourceID, 2, nil,
		transactor.TransactOptions{
			Priority: uint8(2), // fast
			Value:    s.config1.Fee,
		})
	s.Nil(err)

	depositTx, _, err := s.client1.TransactionByHash(context.Background(), *depositTxHash)
	s.Nil(err)
	// check gas price of deposit tx - 140 gwei
	s.Equal(big.NewInt(140000000000), depositTx.GasPrice())

	err = evm.WaitForProposalExecuted(s.client2, s.config2.BridgeAddr)
	s.NotNil(err)

	_, err = erc20Contract2.MintTokens(s.config2.Erc20HandlerAddr, amountToDeposit, transactor.TransactOptions{})
	if err != nil {
		return
	}

	retryTxHash, err := bridgeContract1.Retry(*depositTxHash, txOptions)
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
}
