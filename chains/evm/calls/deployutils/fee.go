package deployutils

import (
	"math/big"

	"github.com/ChainSafe/sygma/chains/evm/calls/contracts/feeHandler"

	"github.com/ChainSafe/sygma/chains/evm/calls/contracts/bridge"

	"github.com/ChainSafe/sygma-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/sygma-core/types"
	"github.com/ethereum/go-ethereum/common"
)

type FeeHandlerSetupConfig struct {
	DestDomainID     uint8
	ResourceID       types.ResourceID
	BridgeContract   *bridge.BridgeContract
	FeeOracleAddress common.Address
	FeePercent       uint16
	FeeGas           uint32
	FeeAmount        *big.Int // For BasicFeeHandler
}
type FeeHandlerDeployResutls struct {
	FeeRouter         *feeHandler.FeeRouter
	FeeHandlerAddress common.Address
	FeeRouterAddress  common.Address
}

func SetupFeeHandlerWithOracle(ethClient EVMClient, t transactor.Transactor, fhc *FeeHandlerSetupConfig) (*FeeHandlerDeployResutls, error) {
	// Deploy
	fr, err := DeployFeeRouter(ethClient, t, *fhc.BridgeContract.ContractAddress())
	if err != nil {
		return nil, err
	}
	fh, err := DeployFeeHandlerWithOracle(ethClient, t, *fhc.BridgeContract.ContractAddress(), *fr.ContractAddress())
	if err != nil {
		return nil, err
	}

	// Setup fee
	//Set FeeHandler on FeeRouter
	_, err = fr.AdminSetResourceHandler(fhc.DestDomainID, fhc.ResourceID, *fh.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}
	// Set FeeOracle address  for FeeHandlers (if required)
	_, err = fh.SetFeeOracle(fhc.FeeOracleAddress, transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}
	// Set fee properties (percentage, gasUsed)
	_, err = fh.SetFeeProperties(fhc.FeeGas, fhc.FeePercent, transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}

	_, err = fhc.BridgeContract.AdminChangeFeeHandler(*fr.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}
	return &FeeHandlerDeployResutls{
		FeeHandlerAddress: *fh.ContractAddress(),
		FeeRouterAddress:  *fr.ContractAddress(),
	}, nil
}

func SetupFeeBasicHandler(ethClient EVMClient, t transactor.Transactor, fhc *FeeHandlerSetupConfig) (*FeeHandlerDeployResutls, error) {
	// Deploy
	fr, err := DeployFeeRouter(ethClient, t, *fhc.BridgeContract.ContractAddress())
	if err != nil {
		return nil, err
	}
	fh, err := DeployBasicFeeHandler(ethClient, t, *fhc.BridgeContract.ContractAddress(), *fr.ContractAddress())
	if err != nil {
		return nil, err
	}

	// Setup fee
	//Set FeeHandler on FeeRouter
	_, err = fr.AdminSetResourceHandler(fhc.DestDomainID, fhc.ResourceID, *fh.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}
	// Set FeeOracle address  for FeeHandlers (if required)
	// Set fee properties (percentage, gasUsed)
	_, err = fh.ChangeFee(fhc.FeeAmount, transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}
	_, err = fhc.BridgeContract.AdminChangeFeeHandler(*fr.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}
	return &FeeHandlerDeployResutls{
		FeeRouter:         fr,
		FeeHandlerAddress: *fh.ContractAddress(),
		FeeRouterAddress:  *fr.ContractAddress(),
	}, nil
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

func DeployBasicFeeHandler(
	ethClient EVMClient, t transactor.Transactor, bridgeContractAddress, feeRouterAddress common.Address,
) (*feeHandler.BasicFeeHandlerContract, error) {
	feeHandlerContract := feeHandler.NewBasicFeeHandlerContract(ethClient, common.Address{}, t)
	_, err := feeHandlerContract.DeployContract(bridgeContractAddress, feeRouterAddress)
	if err != nil {
		return nil, err
	}

	return feeHandlerContract, nil
}
