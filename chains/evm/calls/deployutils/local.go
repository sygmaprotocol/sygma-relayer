package deployutils

import (
	"math/big"

	"github.com/ChainSafe/sygma-core/chains/evm/calls"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/evmgaspricer"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/transactor/signAndSend"
	"github.com/ChainSafe/sygma-core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

var BasicFee = big.NewInt(100000000000)

type EVMClient interface {
	calls.ContractCallerDispatcher
	evmgaspricer.GasPriceClient
}

type BridgeConfig struct {
	BridgeAddr common.Address

	Erc20Addr        common.Address
	Erc20HandlerAddr common.Address
	Erc20ResourceID  types.ResourceID

	Erc20LockReleaseAddr        common.Address
	Erc20LockReleaseHandlerAddr common.Address
	Erc20LockReleaseResourceID  types.ResourceID

	GenericHandlerAddr common.Address
	AssetStoreAddr     common.Address
	GenericResourceID  types.ResourceID

	Erc721Addr        common.Address
	Erc721HandlerAddr common.Address
	Erc721ResourceID  types.ResourceID

	FeeHandlerAddr    common.Address
	FeeRouterAddress  common.Address
	IsBasicFeeHandler bool
	Fee               *big.Int
}

func TestSetupEVMBridge(
	ethClient EVMClient,
	fabric calls.TxFabric,
	domainID uint8,
	destDomainID uint8,
	mintTo common.Address,
) (*BridgeConfig, error) {
	staticGasPricer := evmgaspricer.NewStaticGasPriceDeterminant(ethClient, nil)
	t := signAndSend.NewSignAndSendTransactor(fabric, staticGasPricer, ethClient)

	_, bridgeContract, err := DeployBridgeWithAccessControl(ethClient, t, AdminFunctionHexes, ethClient.From(), domainID)
	if err != nil {
		return nil, err
	}

	bridgeContractAddress := *bridgeContract.ContractAddress()

	_, err = bridgeContract.EndKeygen(MpcAddress, transactor.TransactOptions{})
	if err != nil {
		return nil, err
	}

	erc721Contract, erc721ContractAddress, erc721HandlerContractAddress, err := deployErc721(
		ethClient, t, bridgeContractAddress,
	)
	if err != nil {
		return nil, err
	}

	erc20Contract, erc20ContractAddress, erc20HandlerContractAddress, err := deployErc20(
		ethClient, t, bridgeContractAddress,
	)
	if err != nil {
		return nil, err
	}

	genericHandlerAddress, assetStoreAddress, err := deployGeneric(ethClient, t, bridgeContractAddress)
	if err != nil {
		return nil, err
	}

	erc20LockReleaseContract, erc20LockReleaseContractAddress, err := deployErc20LockRelease(
		ethClient, t, bridgeContractAddress,
	)
	if err != nil {
		return nil, err
	}

	erc20ResourceID := calls.SliceTo32Bytes(common.LeftPadBytes([]byte{0}, 31))
	genericResourceID := calls.SliceTo32Bytes(common.LeftPadBytes([]byte{1}, 31))
	erc721ResourceID := calls.SliceTo32Bytes(common.LeftPadBytes([]byte{2}, 31))
	erc20LockReleaseResourceID := calls.SliceTo32Bytes(common.LeftPadBytes([]byte{3}, 31))

	// Deploy FeeRouter and set it up on the bridge contact
	fr, err := SetupFeeRouter(ethClient, t, bridgeContract)
	if err != nil {
		return nil, err
	}
	fh, err := DeployBasicFeeHandler(ethClient, t, bridgeContractAddress, *fr.ContractAddress())
	if err != nil {
		return nil, err
	}

	_, err = fr.AdminSetResourceHandler(destDomainID, erc20ResourceID, *fh.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}
	_, err = fr.AdminSetResourceHandler(destDomainID, erc721ResourceID, *fh.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}
	_, err = fr.AdminSetResourceHandler(destDomainID, genericResourceID, *fh.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}
	_, err = fr.AdminSetResourceHandler(destDomainID, erc20LockReleaseResourceID, *fh.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}

	conf := &BridgeConfig{
		BridgeAddr: bridgeContractAddress,

		Erc20Addr:        erc20ContractAddress,
		Erc20HandlerAddr: erc20HandlerContractAddress,
		Erc20ResourceID:  erc20ResourceID,

		Erc20LockReleaseAddr:        erc20LockReleaseContractAddress,
		Erc20LockReleaseResourceID:  erc20LockReleaseResourceID,
		Erc20LockReleaseHandlerAddr: erc20HandlerContractAddress,

		GenericHandlerAddr: genericHandlerAddress,
		AssetStoreAddr:     assetStoreAddress,
		GenericResourceID:  genericResourceID,

		Erc721Addr:        erc721ContractAddress,
		Erc721HandlerAddr: erc721HandlerContractAddress,
		Erc721ResourceID:  erc721ResourceID,

		IsBasicFeeHandler: true,
		Fee:               big.NewInt(100000000000),
		FeeHandlerAddr:    *fh.ContractAddress(),
		FeeRouterAddress:  *fr.ContractAddress(),
	}

	err = SetupERC20Handler(bridgeContract, erc20Contract, mintTo, conf)
	if err != nil {
		return nil, err
	}

	err = SetupERC721Handler(bridgeContract, erc721Contract, conf)
	if err != nil {
		return nil, err
	}

	err = SetupGenericHandler(bridgeContract, conf)
	if err != nil {
		return nil, err
	}

	err = SetupERC20LockReleaseHandler(bridgeContract, erc20LockReleaseContract, mintTo, conf)
	if err != nil {
		return nil, err
	}

	_, err = fh.ChangeFee(BasicFee, transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("All deployments and preparations are done")
	return conf, nil
}
