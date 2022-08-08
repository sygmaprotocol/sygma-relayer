// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package deployutils

import (
	"context"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmgaspricer"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/signAndSend"
	"github.com/ChainSafe/chainbridge-core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

var BasicFee = big.NewInt(100000000000)
var OracleFee = uint16(500) // 5% -  multiplied by 100 to not lose precision on contract side
var GasUsed = uint32(100000)
var FeeOracleAddress = common.HexToAddress("0x70B7D7448982b15295150575541D1d3b862f7FE9")

type EVMClient interface {
	calls.ContractCallerDispatcher
	evmgaspricer.GasPriceClient
	ChainID(ctx context.Context) (*big.Int, error)
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

	BasicFeeHandlerAddr      common.Address
	FeeRouterAddress         common.Address
	FeeHandlerWithOracleAddr common.Address
	BasicFee                 *big.Int
	OracleFee                uint16
}

func SetupLocalSygmaRelayer(
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
	fhwo, err := DeployFeeHandlerWithOracle(ethClient, t, bridgeContractAddress, *fr.ContractAddress())
	if err != nil {
		return nil, err
	}

	_, err = fr.AdminSetResourceHandler(destDomainID, erc20ResourceID, *fhwo.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
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
	_, err = fr.AdminSetResourceHandler(destDomainID, erc20LockReleaseResourceID, *fhwo.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
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

		BasicFeeHandlerAddr:      *fh.ContractAddress(),
		FeeRouterAddress:         *fr.ContractAddress(),
		FeeHandlerWithOracleAddr: *fhwo.ContractAddress(),
		OracleFee:                OracleFee,
		BasicFee:                 BasicFee,
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
	_, err = fhwo.SetFeeProperties(GasUsed, OracleFee, transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("All deployments and preparations are done")
	return conf, nil
}
