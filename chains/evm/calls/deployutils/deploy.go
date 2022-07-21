package deployutils

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	"github.com/ChainSafe/sygma-core/chains/evm/calls"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/contracts/erc20"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/evmgaspricer"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/transactor/signAndSend"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/flags"
	"github.com/ChainSafe/sygma-core/types"
	"github.com/ChainSafe/sygma/chains/evm/calls/contracts/accessControlSegregator"
	"github.com/ChainSafe/sygma/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/sygma/chains/evm/calls/contracts/erc20Handler"
	"github.com/ChainSafe/sygma/chains/evm/calls/contracts/feeHandler"
	"github.com/ChainSafe/sygma/chains/evm/calls/util"
	"github.com/ethereum/go-ethereum/common"
)

type EVMClient interface {
	calls.ContractCallerDispatcher
	evmgaspricer.GasPriceClient
}

type DeployResults struct {
	BridgeAddr           common.Address
	AccessControlAddress common.Address
	FeeRouterAddress     common.Address
	FeeHandlerAddress    common.Address

	Erc20Addr        common.Address
	ERC20Sybmol      string
	Erc20HandlerAddr common.Address
	Erc20ResourceID  types.ResourceID

	Fee      uint16
	FeeGas   uint32
	DomainID uint8
}

type DeployConfig struct {
	DeployKey        *ecdsa.PrivateKey
	NodeURL          string
	DomainID         uint8
	ResourceID       string
	FeeOracleAddress common.Address
	DestDomainID     uint8 // Domain ID of the destination network that will be used in FeeRouter for routing feeCalculate requests
	Erc20Symbol      string
	Erc20Name        string
	FeePercent       uint16
	FeeGas           uint32
}

func (r *DeployResults) PrettyPrintDeployResutls() {
	fmt.Printf(`
===============================================
🎉🎉🎉 Sygma Successfully Deployed 🎉🎉🎉

- Chain 1 -
%s

===============================================
`,
		r.PrettyFormatChainInfo(),
	)
}

func (r *DeployResults) PrettyFormatChainInfo() string {
	return fmt.Sprintf(`
Bridge: %s
DomainID: %v
AccessControl: %s
FeeRouter: %s
FeeHandlerWithOracle: %s
ERC20Handler: %s
===============
ERC20: %s
Symbol %s
===============
Fee percent %v
ResourceID %v


`,
		r.BridgeAddr,
		r.DomainID,
		r.AccessControlAddress,
		r.FeeRouterAddress,
		r.FeeHandlerAddress,
		r.Erc20HandlerAddr,
		r.Erc20Addr,
		r.ERC20Sybmol,
		r.Fee,
		r.Erc20ResourceID,
	)
}

// DeployAndInitiallySetupSygma deploys all neccessary smart contracts that in current time deployed on TestNet environment. Should be used for test purposes
func SetupSygma(dc *DeployConfig) (*DeployResults, error) {

	ethClient, err := evmclient.NewEVMClient(dc.NodeURL, dc.DeployKey)
	if err != nil {
		return nil, err
	}
	deployAddress := ethClient.From()
	rID, err := flags.ProcessResourceID(dc.ResourceID)
	if err != nil {
		return nil, err
	}
	gasPricer := evmgaspricer.NewLondonGasPriceClient(ethClient, nil)
	t := signAndSend.NewSignAndSendTransactor(evmtransaction.NewTransaction, gasPricer, ethClient)

	accessControlContract, bridgeContract, err := DeployBridgeWithAccessControl(ethClient, t, AdminFunctionHexes, deployAddress, dc.DomainID)
	if err != nil {
		return nil, err
	}
	erc20HandlerContract, err := DeployErc20Handler(ethClient, t, *bridgeContract.ContractAddress())
	if err != nil {
		return nil, err
	}

	feeResults, err := SetupFeeHandlerWithOracle(ethClient, t, &FeeHandlerSetupConfig{
		dc.DestDomainID,
		rID,
		*bridgeContract.ContractAddress(),
		dc.FeeOracleAddress,
		dc.FeePercent,
		dc.FeeGas,
	})
	if err != nil {
		return nil, err
	}

	// Setting up deployed FeeRouter on the Bridge
	_, err = bridgeContract.AdminChangeFeeHandler(feeResults.FeeHandlerAddress, transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}
	erc20Contract, err := DeployERC20Token(ethClient, t, dc.Erc20Name, dc.Erc20Symbol)
	if err != nil {
		return nil, err
	}

	// Setting up resourceID for ERC20 token
	_, err = bridgeContract.AdminSetResource(*erc20HandlerContract.ContractAddress(), rID, *erc20Contract.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}
	dr := &DeployResults{
		BridgeAddr:           *bridgeContract.ContractAddress(),
		DomainID:             dc.DomainID,
		AccessControlAddress: *accessControlContract.ContractAddress(),
		FeeHandlerAddress:    feeResults.FeeHandlerAddress,
		FeeRouterAddress:     feeResults.FeeRouterAddress,
		Erc20HandlerAddr:     *erc20HandlerContract.ContractAddress(),
		Erc20Addr:            *erc20Contract.ContractAddress(),
		ERC20Sybmol:          dc.Erc20Symbol,
		Erc20ResourceID:      rID,
		Fee:                  dc.FeePercent,
	}
	return dr, nil
}

func DeployErc20Handler(ethClient EVMClient, t transactor.Transactor, bridgeContractAddress common.Address) (*erc20Handler.ERC20HandlerContract, error) {
	erc20HandlerContract := erc20Handler.NewERC20HandlerContract(ethClient, common.Address{}, t)
	_, err := erc20HandlerContract.DeployContract(bridgeContractAddress)
	if err != nil {
		return nil, err
	}
	return erc20HandlerContract, nil
}

// DeployBridgeWithAccessControl Deploys AccessControl contract and Bridge contract. Deployer is made as Admin for all functions to easy the setup. For production this must be changed after deployment.
func DeployBridgeWithAccessControl(
	ethClient EVMClient, t transactor.Transactor, adminFunctionHexes []string, admin common.Address, domainID uint8,
) (*accessControlSegregator.AccessControlSegregatorContract, *bridge.BridgeContract, error) {

	admins := make([]common.Address, len(adminFunctionHexes))
	adminFunctions := make([][4]byte, len(adminFunctionHexes))
	for i, functionHex := range adminFunctionHexes {
		admins[i] = admin
		hexBytes, _ := hex.DecodeString(functionHex)
		adminFunctions[i] = util.SliceTo4Bytes(hexBytes)
	}
	accessControlSegregatorContract := accessControlSegregator.NewAccessControlSegregatorContract(ethClient, common.Address{}, t)
	_, err := accessControlSegregatorContract.DeployContract(
		adminFunctions,
		admins,
	)
	if err != nil {
		return nil, nil, err
	}

	bridgeContract := bridge.NewBridgeContract(ethClient, common.Address{}, t)
	_, err = bridgeContract.DeployContract(domainID, accessControlSegregatorContract.ContractAddress())
	if err != nil {
		return nil, nil, err
	}
	return accessControlSegregatorContract, bridgeContract, nil
}

func DeployFeeRouter(
	ethClient EVMClient, t transactor.Transactor, bridgeContractAddress common.Address,
) (*feeHandler.FeeRouter, error) {
	feeRouterContract := feeHandler.NewFeeRouter(ethClient, common.Address{}, t)
	_, err := feeRouterContract.DeployContract(bridgeContractAddress)
	if err != nil {
		return nil, err
	}
	return feeRouterContract, nil
}

func DeployFeeHandlerWithOracle(
	ethClient EVMClient, t transactor.Transactor, bridgeContractAddress, feeRouterAddress common.Address,
) (*feeHandler.FeeHandlerWithOracleContract, error) {
	feeHandlerContract := feeHandler.NewFeeHandlerWithOracleContract(ethClient, common.Address{}, t)
	_, err := feeHandlerContract.DeployContract(bridgeContractAddress, feeRouterAddress)
	if err != nil {
		return nil, err
	}

	return feeHandlerContract, nil
}

func DeployERC20Token(ethClient EVMClient, t transactor.Transactor, Erc20Name, Erc20Symbol string) (*erc20.ERC20Contract, error) {
	erc20Contract := erc20.NewERC20Contract(ethClient, common.Address{}, t)
	_, err := erc20Contract.DeployContract(Erc20Name, Erc20Symbol)
	if err != nil {
		return nil, err
	}
	return erc20Contract, nil
}

var AdminFunctionHexes = []string{
	"80ae1c28", // adminPauseTransfers
	"ffaac0eb", // adminUnpauseTransfers
	"cb10f215", // adminSetResource
	"5a1ad87c", // adminSetGenericResource
	"8c0c2631", // adminSetBurnable
	"edc20c3c", // adminSetDepositNonce
	"d15ef64e", // adminSetForwarder
	"9d33b6d4", // adminChangeAccessControl
	"8b63aebf", // adminChangeFeeHandler
	"bd2a1820", // adminWithdraw
	"6ba6db6b", // startKeygen
	"d2e5fae9", // endKeygen
	"f5f63b39", // refreshKey
	"a973ec93", //grantAccess
}
