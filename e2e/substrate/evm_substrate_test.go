package substrate

import (
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/erc20"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/signAndSend"
	"github.com/ChainSafe/chainbridge-core/e2e/dummy"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/connection"
	"github.com/ChainSafe/sygma-relayer/e2e/evm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"
	"math/big"
	"testing"
)

const ETHEndpoint = "ws://localhost:8545"
const SubstrateEndpoint = "ws://localhost:9944"

func Test_EVMSubstrate(t *testing.T) {
	// EVM side config
	evmConfig := evm.BridgeConfig{
		BridgeAddr: common.HexToAddress("0x6CdE2Cd82a4F8B74693Ff5e194c19CA08c2d1c68"),

		Erc20Addr:        common.HexToAddress("0xC2D334e2f27A9dB2Ed8C4561De86C1A00EBf6760"),
		Erc20HandlerAddr: common.HexToAddress("0x1ED1d77911944622FCcDDEad8A731fd77E94173e"),
		Erc20ResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{0}, 31)),

		Erc20LockReleaseAddr:        common.HexToAddress("0x1CcB4231f2ff299E1E049De76F0a1D2B415C563A"),
		Erc20LockReleaseHandlerAddr: common.HexToAddress("0x1ED1d77911944622FCcDDEad8A731fd77E94173e"),
		Erc20LockReleaseResourceID:  calls.SliceTo32Bytes(common.LeftPadBytes([]byte{3}, 31)),

		BasicFeeHandlerAddr:      common.HexToAddress("0x78E5b9cEC9aEA29071f070C8cC561F692B3511A6"),
		FeeHandlerWithOracleAddr: common.HexToAddress("0x6A7f23450c9Fc821Bb42Fb9FE77a09aC4b05b026"),
		FeeRouterAddress:         common.HexToAddress("0x9275AC64D6556BE290dd878e5aAA3a5bae08ae0C"),
		BasicFee:                 evm.BasicFee,
		OracleFee:                evm.OracleFee,
	}

	pk, _ := crypto.HexToECDSA("cc2c32b154490f09f70c1c8d4b997238448d649e0777495863db231c4ced3616")
	ethClient, err := evmclient.NewEVMClient(ETHEndpoint, pk)
	if err != nil {
		panic(err)
	}
	gasPricer := dummy.NewStaticGasPriceDeterminant(ethClient, nil)

	substrateClient, err := connection.NewSubstrateConnection(SubstrateEndpoint)
	if err != nil {
		panic(err)
	}
	_, err = substrateClient.GetBlockLatest()
	if err != nil {
		panic(err)
	}

	suite.Run(
		t,
		NewEVMSubstrateTestSuite(
			evmtransaction.NewTransaction,
			ethClient,
			substrateClient,
			gasPricer,
			evmConfig,
		),
	)
}

func NewEVMSubstrateTestSuite(
	fabric calls.TxFabric,
	evmClient evm.EVMClient,
	substrateClient *connection.Connection,
	gasPricer calls.GasPricer,
	evmConfig evm.BridgeConfig,
) *IntegrationTestSuite {
	return &IntegrationTestSuite{
		fabric:          fabric,
		evmClient:       evmClient,
		substrateClient: substrateClient,
		gasPricer:       gasPricer,
		evmConfig:       evmConfig,
	}
}

type IntegrationTestSuite struct {
	suite.Suite
	fabric          calls.TxFabric
	evmClient       evm.EVMClient
	substrateClient *connection.Connection
	gasPricer       calls.GasPricer
	evmConfig       evm.BridgeConfig
}

func (s *IntegrationTestSuite) SetupSuite() {
	// EVM side preparation
	transactor1 := signAndSend.NewSignAndSendTransactor(s.fabric, s.gasPricer, s.evmClient)
	erc20Contract := erc20.NewERC20Contract(s.evmClient, s.evmConfig.Erc20Addr, transactor1)
	mintTo := s.evmClient.From()
	amountToMint := big.NewInt(0).Mul(big.NewInt(50000), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))
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

	erc20LRContract := erc20.NewERC20Contract(s.evmClient, s.evmConfig.Erc20LockReleaseAddr, transactor1)
	_, err = erc20LRContract.MintTokens(mintTo, amountToMint, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}
	// Approving tokens
	_, err = erc20LRContract.ApproveTokens(s.evmConfig.Erc20HandlerAddr, amountToApprove, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}

	// Substrate side preparation
	// 1. change fungible asset ResourceId on pallet to be 0000000000000000000000000000000000000000000000000000000000000000, which is matching with Erc20ResourceID
	// 2. change DestVerifyingContractAddress on pallet to be deployed bridge address, assuming both source and dest bridge address in e2e test are the same
	// 3. set mpc key in pallet
	// 4. set basic fee in pallet
	// 5. make sure pallet is unpaused
}

func (s *IntegrationTestSuite) Test_Erc20Deposit_EVM_to_Substrate() {
	// TODO: implement me!
}

func (s *IntegrationTestSuite) Test_RetryDeposit_EVM_to_Substrate() {
	// TODO: implement me!
}

func (s *IntegrationTestSuite) Test_PausePalletDeposit_EVM_to_Substrate() {
	// TODO: implement me!
}

func (s *IntegrationTestSuite) Test_DepositSubstrateAsset_Substrate_to_EVM() {
	// TODO: implement me!
}

func (s *IntegrationTestSuite) Test_RetryDeposit_Substrate_to_EVM() {
	// TODO: implement me!
}

func (s *IntegrationTestSuite) Test_PausePalletDeposit_Substrate_to_EVM() {
	// TODO: implement me!
}
