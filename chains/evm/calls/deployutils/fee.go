package deployutils

import (
	"github.com/ChainSafe/sygma-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/sygma-core/types"
	"github.com/ethereum/go-ethereum/common"
)

type FeeHandlerSetupConfig struct {
	DestDomainID          uint8
	ResourceID            types.ResourceID
	BridgeContractAddress common.Address
	FeeOracleAddress      common.Address
	FeePercent            uint16
	FeeGas                uint32
}
type FeeHandlerDeployResutls struct {
	FeeHandlerAddress common.Address
	FeeRouterAddress  common.Address
}

func SetupFeeHandlerWithOracle(ethClient EVMClient, t transactor.Transactor, fhc *FeeHandlerSetupConfig) (*FeeHandlerDeployResutls, error) {
	// Deploy
	fr, err := DeployFeeRouter(ethClient, t, fhc.BridgeContractAddress)
	if err != nil {
		return nil, err
	}
	fh, err := DeployFeeHandlerWithOracle(ethClient, t, fhc.BridgeContractAddress, *fr.ContractAddress())

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
	return &FeeHandlerDeployResutls{
		FeeHandlerAddress: *fh.ContractAddress(),
		FeeRouterAddress:  *fr.ContractAddress(),
	}, nil
}
