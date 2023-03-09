// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package evm_test

import (
	"context"
	"math/big"
	"math/rand"
	"sync"
	"testing"
	"time"

	substrateTypes "github.com/centrifuge/go-substrate-rpc-client/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/centrifuge"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/erc20"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/erc721"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/signAndSend"
	"github.com/ChainSafe/chainbridge-core/e2e/dummy"
	"github.com/ChainSafe/chainbridge-core/keystore"

	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/sygma-relayer/e2e/evm"
)

type TestClient interface {
	evm.EVMClient
	LatestBlock() (*big.Int, error)
	CodeAt(ctx context.Context, contractAddress common.Address, block *big.Int) ([]byte, error)
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]types.Log, error)
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
}

const ETHEndpoint1 = "ws://localhost:8545"
const ETHEndpoint2 = "ws://localhost:8547"

// Alice key is used by the relayer, Charlie key is used as admin and depositer
func Test_EVM2EVM(t *testing.T) {
	rand.Seed(time.Now().Unix())
	config := evm.BridgeConfig{
		BridgeAddr: common.HexToAddress("0x6CdE2Cd82a4F8B74693Ff5e194c19CA08c2d1c68"),

		Erc20Addr:        common.HexToAddress("0x37356a2B2EbF65e5Ea18BD93DeA6869769099739"),
		Erc20HandlerAddr: common.HexToAddress("0x02091EefF969b33A5CE8A729DaE325879bf76f90"),
		Erc20ResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{0}, 31)),

		Erc20LockReleaseAddr:        common.HexToAddress("0x78E5b9cEC9aEA29071f070C8cC561F692B3511A6"),
		Erc20LockReleaseHandlerAddr: common.HexToAddress("0x02091EefF969b33A5CE8A729DaE325879bf76f90"),
		Erc20LockReleaseResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{3}, 31)),

		Erc721Addr:        common.HexToAddress("0xE54Dc792c226AEF99D6086527b98b36a4ADDe56a"),
		Erc721HandlerAddr: common.HexToAddress("0xC2D334e2f27A9dB2Ed8C4561De86C1A00EBf6760"),
		Erc721ResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{2}, 31)),

		GenericHandlerAddr: common.HexToAddress("0xF28c11CB14C6d2B806f99EA8b138F65e74a1Ed66"),
		GenericResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{1}, 31)),
		AssetStoreAddr:     common.HexToAddress("0xB1387B365AE7294Ea13bad9db83436e671DD16Ba"),

		PermissionlessGenericHandlerAddr: common.HexToAddress("0xE837D42dd3c685839a418886f418769BDD23546b"),
		PermissionlessGenericResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{5}, 31)),

		BasicFeeHandlerAddr:      common.HexToAddress("0x8dA96a8C2b2d3e5ae7e668d0C94393aa8D5D3B94"),
		FeeHandlerWithOracleAddr: common.HexToAddress("0x30d704A60037DfE54e7e4D242Ea0cBC6125aE497"),
		FeeRouterAddress:         common.HexToAddress("0x1CcB4231f2ff299E1E049De76F0a1D2B415C563A"),
		BasicFee:                 evm.BasicFee,
		OracleFee:                evm.OracleFee,
	}

	pk, _ := crypto.HexToECDSA("cc2c32b154490f09f70c1c8d4b997238448d649e0777495863db231c4ced3616")
	ethClient1, err := evmclient.NewEVMClient(ETHEndpoint1, pk)
	if err != nil {
		panic(err)
	}
	gasPricer1 := dummy.NewStaticGasPriceDeterminant(ethClient1, nil)

	ethClient2, err := evmclient.NewEVMClient(ETHEndpoint2, pk)
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
	config1, config2 evm.BridgeConfig,
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
	config1    evm.BridgeConfig
	config2    evm.BridgeConfig
}

func (s *IntegrationTestSuite) SetupSuite() {
	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)
	erc20Contract := erc20.NewERC20Contract(s.client1, s.config1.Erc20Addr, transactor1)
	mintTo := s.client1.From()
	amountToMint := big.NewInt(0).Mul(big.NewInt(5000000000000000), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))

	amountToApprove := big.NewInt(0).Mul(big.NewInt(100000), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))
	_, err := erc20Contract.MintTokens(mintTo, amountToMint, transactor.TransactOptions{})

	if err != nil {
		panic(err)
	}

	_, err = erc20Contract.MintTokens(s.config1.Erc20HandlerAddr, amountToMint, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}
	// Approving tokens
	_, err = erc20Contract.ApproveTokens(s.config1.Erc20HandlerAddr, amountToApprove, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}
	_, err = erc20Contract.ApproveTokens(s.config1.FeeHandlerWithOracleAddr, amountToApprove, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}

	erc20LRContract := erc20.NewERC20Contract(s.client1, s.config1.Erc20LockReleaseAddr, transactor1)
	_, err = erc20LRContract.MintTokens(mintTo, amountToMint, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}

	erc20LRContract2 := erc20.NewERC20Contract(s.client2, s.config2.Erc20LockReleaseAddr, transactor2)
	_, err = erc20LRContract2.MintTokens(s.config2.Erc20LockReleaseHandlerAddr, amountToMint.Mul(amountToMint, big.NewInt(10000)), transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}

	// Approving tokens
	_, err = erc20LRContract.ApproveTokens(s.config1.Erc20HandlerAddr, amountToApprove, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}
}

var amountToDeposit = big.NewInt(100)

func (s *IntegrationTestSuite) Test_Erc20Deposit() {
	pk, _ := crypto.HexToECDSA("cc2c32b154490f09f70c1c8d4b997238448d649e0777495863db231c4ced3616")
	dstAddr := crypto.PubkeyToAddress(pk.PublicKey)

	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	erc20Contract1 := erc20.NewERC20Contract(s.client1, s.config1.Erc20Addr, transactor1)
	bridgeContract1 := bridge.NewBridgeContract(s.client1, s.config1.BridgeAddr, transactor1)

	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)
	erc20Contract2 := erc20.NewERC20Contract(s.client2, s.config2.Erc20Addr, transactor2)

	senderBalBefore, err := erc20Contract1.GetBalance(crypto.PubkeyToAddress(pk.PublicKey))
	s.Nil(err)
	destBalanceBefore, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)

	var feeOracleSignature = "8167ba25cf7a08a43aae68576b71f0e42b6281a379a245a8be016c5b16d6227d3941da8f50c7b99763493d6e6f4f36e290ecd9bacca927a2f1b5f157cbe67b171b"
	var feeDataHash = "00000000000000000000000000000000000000000000000000011f667bbfc00000000000000000000000000000000000000000000000000006bb5a99744a9000000000000000000000000000000000000000000000000000000000174876e80000000000000000000000000000000000000000000000000000000000698d283a0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	var feeData = evm.ConstructFeeData(feeOracleSignature, feeDataHash, amountToDeposit)

	depositTxHash, err := bridgeContract1.Erc20Deposit(dstAddr.Bytes(), amountToDeposit, s.config1.Erc20ResourceID, 2, feeData,
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
}

func (s *IntegrationTestSuite) Test_Erc721Deposit() {
	tokenId := big.NewInt(int64(rand.Int()))
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

func (s *IntegrationTestSuite) Test_PermissionlessGenericDeposit() {
	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)

	bridgeContract1 := bridge.NewBridgeContract(s.client1, s.config1.BridgeAddr, transactor1)
	assetStoreContract2 := centrifuge.NewAssetStoreContract(s.client2, s.config2.AssetStoreAddr, transactor2)

	hash, _ := substrateTypes.GetHash(substrateTypes.NewI64(int64(rand.Int())))
	functionSig := string(crypto.Keccak256([]byte("storeWithDepositor(address,bytes32,address)"))[:4])
	contractAddress := assetStoreContract2.ContractAddress()
	maxFee := big.NewInt(200000)
	depositor := s.client1.From()
	var metadata []byte
	metadata = append(metadata, common.LeftPadBytes(hash[:], 32)...)
	metadata = append(metadata, common.LeftPadBytes(depositor.Bytes(), 32)...)

	_, err := bridgeContract1.PermissionlessGenericDeposit(metadata, functionSig, contractAddress, &depositor, maxFee, s.config1.PermissionlessGenericResourceID, 2, nil, transactor.TransactOptions{
		Value: s.config1.BasicFee,
	})
	s.Nil(err)

	err = evm.WaitForProposalExecuted(s.client2, s.config2.BridgeAddr)
	s.Nil(err)

	exists, err := assetStoreContract2.IsCentrifugeAssetStored(hash)
	s.Nil(err)
	s.Equal(true, exists)
}

func (s *IntegrationTestSuite) Test_RetryDeposit() {
	dstAddr := keystore.TestKeyRing.EthereumKeys[keystore.BobKey].CommonAddress()
	amountToMint := big.NewInt(0).Mul(big.NewInt(250), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))

	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	bridgeContract1 := bridge.NewBridgeContract(s.client1, s.config1.BridgeAddr, transactor1)

	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)
	erc20Contract2 := erc20.NewERC20Contract(s.client2, s.config2.Erc20LockReleaseAddr, transactor2)

	destBalanceBefore, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)

	depositTxHash, err := bridgeContract1.Erc20Deposit(dstAddr.Bytes(), amountToDeposit, s.config1.Erc20LockReleaseResourceID, 2, nil,
		transactor.TransactOptions{
			Priority: uint8(2), // fast
			Value:    s.config1.BasicFee,
		})
	s.Nil(err)

	log.Debug().Msgf("deposit hash %s", depositTxHash.Hex())

	_, _, err = s.client1.TransactionByHash(context.Background(), *depositTxHash)
	s.Nil(err)
	time.Sleep(time.Second * 15)

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
	time.Sleep(time.Second * 15)

	destBalanceAfter, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)
	//Balance has increased
	s.Equal(1, destBalanceAfter.Cmp(destBalanceBefore))
}

func (s *IntegrationTestSuite) Test_MultipleDeposits() {
	dstAddr := keystore.TestKeyRing.EthereumKeys[keystore.BobKey].CommonAddress()

	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric1, s.gasPricer1, s.client1)
	bridgeContract1 := bridge.NewBridgeContract(s.client1, s.config1.BridgeAddr, transactor1)

	transactor2 := signAndSend.NewSignAndSendTransactor(s.fabric2, s.gasPricer2, s.client2)
	erc20Contract2 := erc20.NewERC20Contract(s.client2, s.config2.Erc20Addr, transactor2)

	destBalanceBefore, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)

	var feeOracleSignature = "8167ba25cf7a08a43aae68576b71f0e42b6281a379a245a8be016c5b16d6227d3941da8f50c7b99763493d6e6f4f36e290ecd9bacca927a2f1b5f157cbe67b171b"
	var feeDataHash = "00000000000000000000000000000000000000000000000000011f667bbfc00000000000000000000000000000000000000000000000000006bb5a99744a9000000000000000000000000000000000000000000000000000000000174876e80000000000000000000000000000000000000000000000000000000000698d283a0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	var feeData = evm.ConstructFeeData(feeOracleSignature, feeDataHash, amountToDeposit)

	numOfDeposits := 25

	var wg sync.WaitGroup
	for i := 0; i < numOfDeposits; i++ {
		go func() {
			_, err := bridgeContract1.Erc20Deposit(dstAddr.Bytes(), amountToDeposit, s.config1.Erc20ResourceID, 2, feeData,
				transactor.TransactOptions{
					Priority: uint8(2), // fast
				})
			wg.Add(1)
			defer wg.Done()
			s.Nil(err)
		}()
	}
	wg.Wait()
	time.Sleep(30 * time.Second)

	err = evm.WaitForProposalExecuted(s.client2, s.config2.BridgeAddr)
	s.Nil(err)

	destBalanceAfter, err := erc20Contract2.GetBalance(dstAddr)
	s.Nil(err)
	//Balance has increased
	s.Equal(1, destBalanceAfter.Cmp(destBalanceBefore))
}
