package substrate_test

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/deposit"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/client"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/connection"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	substrateTypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/core/types"

	"math/big"
	"testing"

	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/bridge"
	"github.com/rs/zerolog/log"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/erc20"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/signAndSend"
	"github.com/ChainSafe/chainbridge-core/e2e/dummy"
	"github.com/ChainSafe/sygma-relayer/e2e/evm"
	"github.com/ChainSafe/sygma-relayer/e2e/substrate"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"
)

const ETHEndpoint = "ws://localhost:8545"
const SubstrateEndpoint = "ws://localhost:9944"

var substratePK = signature.KeyringPair{
	URI:       "//Alice",
	PublicKey: []byte{0xd4, 0x35, 0x93, 0xc7, 0x15, 0xfd, 0xd3, 0x1c, 0x61, 0x14, 0x1a, 0xbd, 0x4, 0xa9, 0x9f, 0xd6, 0x82, 0x2c, 0x85, 0x58, 0x85, 0x4c, 0xcd, 0xe3, 0x9a, 0x56, 0x84, 0xe7, 0xa5, 0x6d, 0xa2, 0x7d},
	Address:   "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
}

type TestClient interface {
	evm.EVMClient
	LatestBlock() (*big.Int, error)
	CodeAt(ctx context.Context, contractAddress common.Address, block *big.Int) ([]byte, error)
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]types.Log, error)
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
}

func Test_EVMSubstrate(t *testing.T) {
	// EVM side config
	evmConfig := evm.BridgeConfig{
		BridgeAddr: common.HexToAddress("0x6CdE2Cd82a4F8B74693Ff5e194c19CA08c2d1c68"),

		Erc20Addr:        common.HexToAddress("0x1D20a9AcDBE9466E7C07859Cf17fB3A93f010c8D"),
		Erc20HandlerAddr: common.HexToAddress("0x02091EefF969b33A5CE8A729DaE325879bf76f90"),
		Erc20ResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{0}, 31)),

		Erc20LockReleaseAddr:        common.HexToAddress("0x78E5b9cEC9aEA29071f070C8cC561F692B3511A6"),
		Erc20LockReleaseHandlerAddr: common.HexToAddress("0x02091EefF969b33A5CE8A729DaE325879bf76f90"),
		Erc20LockReleaseResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{3}, 31)),

		BasicFeeHandlerAddr:      common.HexToAddress("0x8dA96a8C2b2d3e5ae7e668d0C94393aa8D5D3B94"),
		FeeHandlerWithOracleAddr: common.HexToAddress("0x30d704A60037DfE54e7e4D242Ea0cBC6125aE497"),
		FeeRouterAddress:         common.HexToAddress("0x1CcB4231f2ff299E1E049De76F0a1D2B415C563A"),
		BasicFee:                 evm.BasicFee,
		OracleFee:                evm.OracleFee,
	}

	pk, _ := crypto.HexToECDSA("cc2c32b154490f09f70c1c8d4b997238448d649e0777495863db231c4ced3616")
	ethClient, err := evmclient.NewEVMClient(ETHEndpoint, pk)
	if err != nil {
		panic(err)
	}
	gasPricer := dummy.NewStaticGasPriceDeterminant(ethClient, nil)

	substrateClient, err := client.NewSubstrateClient(SubstrateEndpoint, &substratePK)
	if err != nil {
		panic(err)
	}
	substrateConnection, err := connection.NewSubstrateConnection(SubstrateEndpoint)
	if err != nil {
		panic(err)
	}
	suite.Run(
		t,
		NewEVMSubstrateTestSuite(
			evmtransaction.NewTransaction,
			ethClient,
			substrateClient,
			substrateConnection,
			gasPricer,
			evmConfig,
		),
	)
}

func NewEVMSubstrateTestSuite(
	fabric calls.TxFabric,
	evmClient TestClient,
	substrateClient *client.SubstrateClient,
	substrateConnection *connection.Connection,
	gasPricer calls.GasPricer,
	evmConfig evm.BridgeConfig,
) *IntegrationTestSuite {
	return &IntegrationTestSuite{
		fabric:              fabric,
		evmClient:           evmClient,
		substrateClient:     substrateClient,
		substrateConnection: substrateConnection,
		gasPricer:           gasPricer,
		evmConfig:           evmConfig,
	}
}

type IntegrationTestSuite struct {
	suite.Suite
	fabric              calls.TxFabric
	evmClient           TestClient
	substrateClient     *client.SubstrateClient
	substrateConnection *connection.Connection
	gasPricer           calls.GasPricer
	evmConfig           evm.BridgeConfig
}

func (s *IntegrationTestSuite) SetupSuite() {
	// EVM side preparation
	evmTransactor := signAndSend.NewSignAndSendTransactor(s.fabric, s.gasPricer, s.evmClient)
	erc20Contract := erc20.NewERC20Contract(s.evmClient, s.evmConfig.Erc20Addr, evmTransactor)
	mintTo := s.evmClient.From()
	amountToMint := big.NewInt(0).Mul(big.NewInt(5000000000000000), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))
	amountToApprove := big.NewInt(0).Mul(big.NewInt(100000), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))
	_, err := erc20Contract.MintTokens(mintTo, amountToMint, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}
	_, err = erc20Contract.MintTokens(s.evmConfig.Erc20HandlerAddr, amountToMint, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}
	// Approving tokens
	_, err = erc20Contract.ApproveTokens(s.evmConfig.Erc20HandlerAddr, amountToApprove, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}
	_, err = erc20Contract.ApproveTokens(s.evmConfig.FeeHandlerWithOracleAddr, amountToApprove, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}

	erc20LRContract := erc20.NewERC20Contract(s.evmClient, s.evmConfig.Erc20LockReleaseAddr, evmTransactor)
	_, err = erc20LRContract.MintTokens(mintTo, amountToMint, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}
}

func (s *IntegrationTestSuite) Test_Erc20Deposit_Substrate_to_EVM() {

	meta := s.substrateConnection.GetMetadata()
	key, _ := substrateTypes.CreateStorageKey(&meta, "System", "Account", substratePK.PublicKey)
	var accountInfo1 substrateTypes.AccountInfo
	s.substrateConnection.RPC.State.GetStorageLatest(key, &accountInfo1)
	senderBalanceBefore := accountInfo1.Data.Free

	pk, _ := crypto.HexToECDSA("cc2c32b154490f09f70c1c8d4b997238448d649e0777495863db231c4ced3616")
	dstAddr := crypto.PubkeyToAddress(pk.PublicKey)
	fmt.Println(dstAddr)
	transactor := signAndSend.NewSignAndSendTransactor(s.fabric, s.gasPricer, s.evmClient)

	erc20Contract := erc20.NewERC20Contract(s.evmClient, s.evmConfig.Erc20Addr, transactor)

	destBalanceBefore, err := erc20Contract.GetBalance(dstAddr)
	s.Nil(err)

	res := common.Hex2Bytes("0x95ECF5ae000e0fe0e0dE63aDE9b7D82a372038b4")
	fmt.Println(res)
	assetLocation := [3]substrateTypes.JunctionV1{
		substrateTypes.JunctionV1{
			IsParachain: true,
			ParachainID: substrateTypes.NewUCompact(big.NewInt(2004)),
		},
		substrateTypes.JunctionV1{
			IsGeneralKey: true,
			GeneralKey:   []substrateTypes.U8("sygma"),
		},
		substrateTypes.JunctionV1{
			IsGeneralKey: true,
			GeneralKey:   []substrateTypes.U8("usdc"),
		},
	}
	multiLocation := substrateTypes.MultiLocationV1{
		Parents: 1,
		Interior: substrateTypes.JunctionsV1{
			IsX3: true,
			X3:   assetLocation,
		},
	}

	multiAsset := substrateTypes.MultiAssetV1{
		ID: substrateTypes.AssetID{
			IsConcrete:    true,
			MultiLocation: multiLocation,
		},
		Fungibility: substrateTypes.Fungibility{
			IsFungible: true,
			Amount:     substrateTypes.NewUCompact(big.NewInt(20000000000000)),
		},
	}
	reciever := []substrateTypes.U8{92, 31, 89, 97, 105, 107, 173, 46, 115, 247, 52, 23, 240, 126, 245, 92, 98, 162, 220, 91}
	dst := [2]JunctionV1{
		JunctionV1{
			IsGeneralKey: true,
			GeneralKey:   reciever,
		},
		JunctionV1{
			IsGeneralIndex: true,
			GeneralIndex:   substrateTypes.NewUCompact(big.NewInt(1)),
		},
	}
	dest := MultiLocationV1{
		Parents: 0,
		Interior: JunctionsV1{
			IsX2: true,
			X2:   dst,
		},
	}
	hsh, err := s.substrateClient.Transact(s.substrateConnection, "SygmaBridge.deposit", multiAsset, dest)
	s.Nil(err)
	fmt.Println("hshshhshs")
	fmt.Println(hsh.Hex())
	err = evm.WaitForProposalExecuted(s.evmClient, s.evmConfig.BridgeAddr)
	s.Nil(err)

	meta = s.substrateConnection.GetMetadata()
	key, _ = substrateTypes.CreateStorageKey(&meta, "System", "Account", substratePK.PublicKey)
	var accountInfo2 substrateTypes.AccountInfo

	s.substrateConnection.RPC.State.GetStorageLatest(key, &accountInfo2)
	senderBalanceAfter := accountInfo2.Data.Free

	// balance of sender has decreased
	s.Equal(1, senderBalanceBefore.Int.Cmp(senderBalanceAfter.Int))
	destBalanceAfter, err := erc20Contract.GetBalance(dstAddr)

	s.Nil(err)
	//Balance has increased
	s.Equal(1, destBalanceAfter.Cmp(destBalanceBefore))
}

var amountToDeposit = big.NewInt(100000000000000)

func (s *IntegrationTestSuite) Test_Erc20Deposit_EVM_to_Substrate() {
	pk, _ := crypto.HexToECDSA("cc2c32b154490f09f70c1c8d4b997238448d649e0777495863db231c4ced3616")
	dstAddr := substratePK.PublicKey

	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric, s.gasPricer, s.evmClient)
	erc20Contract1 := erc20.NewERC20Contract(s.evmClient, s.evmConfig.Erc20Addr, transactor1)
	bridgeContract1 := bridge.NewBridgeContract(s.evmClient, s.evmConfig.BridgeAddr, transactor1)

	senderBalBefore, err := erc20Contract1.GetBalance(crypto.PubkeyToAddress(pk.PublicKey))
	s.Nil(err)

	meta := s.substrateConnection.GetMetadata()
	var acc Account
	var assetId uint32 = 2000
	assetIdSerialized := make([]byte, 4)
	binary.LittleEndian.PutUint32(assetIdSerialized, assetId)

	key, _ := substrateTypes.CreateStorageKey(&meta, "Assets", "Account", assetIdSerialized, dstAddr)
	s.substrateConnection.RPC.State.GetStorageLatest(key, &acc)
	destBalanceBefore := acc.Balance
	s.Nil(err)

	var feeOracleSignature = "8167ba25cf7a08a43aae68576b71f0e42b6281a379a245a8be016c5b16d6227d3941da8f50c7b99763493d6e6f4f36e290ecd9bacca927a2f1b5f157cbe67b171b"
	var feeDataHash = "00000000000000000000000000000000000000000000000000011f667bbfc00000000000000000000000000000000000000000000000000006bb5a99744a9000000000000000000000000000000000000000000000000000000000174876e80000000000000000000000000000000000000000000000000000000000698d283a0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	var feeData = evm.ConstructFeeData(feeOracleSignature, feeDataHash, amountToDeposit)
	dt := deposit.ConstructErc20DepositData(dstAddr, amountToDeposit)

	opts := transactor.TransactOptions{
		Priority: uint8(2), // fast
	}
	depositTxHash, err := bridgeContract1.ExecuteTransaction(
		"deposit",
		opts,
		uint8(2), s.evmConfig.Erc20ResourceID, dt, feeData,
	)
	s.Nil(err)

	log.Debug().Msgf("deposit hash %s", depositTxHash.Hex())

	depositTx, _, err := s.evmClient.TransactionByHash(context.Background(), *depositTxHash)
	s.Nil(err)
	// check gas price of deposit tx - 140 gwei
	s.Equal(big.NewInt(140000000000), depositTx.GasPrice())

	err = substrate.WaitForProposalExecuted(s.substrateConnection, destBalanceBefore, dstAddr)
	s.Nil(err)
	senderBalAfter, err := erc20Contract1.GetBalance(s.evmClient.From())
	s.Nil(err)
	s.Equal(-1, senderBalAfter.Cmp(senderBalBefore))

	key, _ = substrateTypes.CreateStorageKey(&meta, "Assets", "Account", assetIdSerialized, dstAddr)
	s.substrateConnection.RPC.State.GetStorageLatest(key, &acc)
	destBalanceAfter := acc.Balance

	//Balance has increased
	s.Equal(1, destBalanceAfter.Int.Cmp(destBalanceBefore.Int))
}

type Account struct {
	Balance substrateTypes.U128
}

type MultiLocationV1 struct {
	Parents  substrateTypes.U8
	Interior JunctionsV1
}
type JunctionsV1 struct {
	IsHere bool

	IsX1 bool
	X1   JunctionV1

	IsX2 bool
	X2   [2]JunctionV1

	IsX3 bool
	X3   [3]JunctionV1

	IsX4 bool
	X4   [4]JunctionV1

	IsX5 bool
	X5   [5]JunctionV1

	IsX6 bool
	X6   [6]JunctionV1

	IsX7 bool
	X7   [7]JunctionV1

	IsX8 bool
	X8   [8]JunctionV1
}

type JunctionV1 struct {
	IsParachain bool
	ParachainID substrateTypes.UCompact

	IsAccountID32        bool
	AccountID32NetworkID substrateTypes.NetworkID
	AccountID            []substrateTypes.U8

	IsAccountIndex64        bool
	AccountIndex64NetworkID substrateTypes.NetworkID
	AccountIndex            substrateTypes.U64

	IsAccountKey20        bool
	AccountKey20NetworkID substrateTypes.NetworkID
	AccountKey            []substrateTypes.U8

	IsPalletInstance bool
	PalletIndex      substrateTypes.U8

	IsGeneralIndex bool
	GeneralIndex   substrateTypes.UCompact // in the original package this is wrong type (U128)

	IsGeneralKey bool
	GeneralKey   []substrateTypes.U8

	IsOnlyChild bool

	IsPlurality bool
	BodyID      substrateTypes.BodyID
	BodyPart    substrateTypes.BodyPart
}
