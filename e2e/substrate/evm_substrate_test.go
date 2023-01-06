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
	// Approving tokens
	_, err = erc20LRContract.ApproveTokens(s.evmConfig.Erc20HandlerAddr, amountToApprove, transactor.TransactOptions{})
	if err != nil {
		panic(err)
	}
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
