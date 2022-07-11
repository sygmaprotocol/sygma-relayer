// Copyright 2021 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package local

import (
	"math/big"

	"github.com/ChainSafe/chainbridge-hub/chains/evm/calls/contracts/accessControlSegregator"
	"github.com/ChainSafe/chainbridge-hub/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/chainbridge-hub/chains/evm/calls/contracts/feeHandler"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/centrifuge"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/erc20"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/erc721"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/generic"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmgaspricer"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/signAndSend"
	"github.com/ChainSafe/chainbridge-core/keystore"
	"github.com/ChainSafe/chainbridge-core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

var AliceKp = keystore.TestKeyRing.EthereumKeys[keystore.AliceKey]
var BobKp = keystore.TestKeyRing.EthereumKeys[keystore.BobKey]
var EveKp = keystore.TestKeyRing.EthereumKeys[keystore.EveKey]

var (
	MpcAddress = common.HexToAddress("0x1c5541A79AcC662ab2D2647F3B141a3B7Cdb2Ae4")
)

type BridgeConfig struct {
	BridgeAddr common.Address

	Erc20Addr        common.Address
	Erc20HandlerAddr common.Address
	Erc20ResourceID  types.ResourceID

	GenericHandlerAddr common.Address
	AssetStoreAddr     common.Address
	GenericResourceID  types.ResourceID

	Erc721Addr        common.Address
	Erc721HandlerAddr common.Address
	Erc721ResourceID  types.ResourceID

	FeeHandlerAddr    common.Address
	IsBasicFeeHandler bool
	Fee               *big.Int
}

type EVMClient interface {
	calls.ContractCallerDispatcher
	evmgaspricer.GasPriceClient
}

func SetupEVMBridge(
	ethClient EVMClient,
	fabric calls.TxFabric,
	domainID uint8,
	mintTo common.Address,
) (BridgeConfig, error) {
	staticGasPricer := evmgaspricer.NewStaticGasPriceDeterminant(ethClient, nil)
	t := signAndSend.NewSignAndSendTransactor(fabric, staticGasPricer, ethClient)

	accessControlSegregatorContract := accessControlSegregator.NewAccessControlSegregatorContract(ethClient, common.Address{}, t)
	adminFunctions := []string{
		"0x80ae1c28", // adminPauseTransfers
		"0xffaac0eb", // adminUnpauseTransfers
		"0xcb10f215", // adminSetResource
		"0x5a1ad87c", // adminSetGenericResource
		"0x8c0c2631", // adminSetBurnable
		"0xedc20c3c", // adminSetDepositNonce
		"0xd15ef64e", // adminSetForwarder
		"0x9d33b6d4", // adminChangeAccessControl
		"0x8b63aebf", // adminChangeFeeHandler
		"0xbd2a1820", // adminWithdraw
		"0x6ba6db6b", // startKeygen
		"0xd2e5fae9", // endKeygen
		"0xf5f63b39", // refreshKey
	}
	admins := make([]common.Address, len(adminFunctions))
	for i, _ := range adminFunctions {
		admins[i] = ethClient.From()
	}
	_, err := accessControlSegregatorContract.DeployContract(
		adminFunctions,
		admins,
	)
	if err != nil {
		return BridgeConfig{}, err
	}

	bridgeContract := bridge.NewBridgeContract(ethClient, common.Address{}, t)
	bridgeContractAddress, err := bridgeContract.DeployContract(domainID, accessControlSegregatorContract.ContractAddress())
	if err != nil {
		return BridgeConfig{}, err
	}
	_, err = bridgeContract.EndKeygen(MpcAddress, transactor.TransactOptions{})
	if err != nil {
		return BridgeConfig{}, err
	}

	erc721Contract, erc721ContractAddress, erc721HandlerContractAddress, err := deployErc721(
		ethClient, t, bridgeContractAddress,
	)
	if err != nil {
		return BridgeConfig{}, err
	}

	erc20Contract, erc20ContractAddress, erc20HandlerContractAddress, err := deployErc20(
		ethClient, t, bridgeContractAddress,
	)

	if err != nil {
		return BridgeConfig{}, err
	}

	genericHandlerAddress, assetStoreAddress, err := deployGeneric(ethClient, t, bridgeContractAddress)
	if err != nil {
		return BridgeConfig{}, err
	}

	feeHandlerContract, err := deployFeeHandler(ethClient, t, bridgeContractAddress)
	if err != nil {
		return BridgeConfig{}, err
	}

	erc20ResourceID := calls.SliceTo32Bytes(common.LeftPadBytes([]byte{0}, 31))
	genericResourceID := calls.SliceTo32Bytes(common.LeftPadBytes([]byte{1}, 31))
	erc721ResourceID := calls.SliceTo32Bytes(common.LeftPadBytes([]byte{2}, 31))

	conf := BridgeConfig{
		BridgeAddr: bridgeContractAddress,

		Erc20Addr:        erc20ContractAddress,
		Erc20HandlerAddr: erc20HandlerContractAddress,
		Erc20ResourceID:  erc20ResourceID,

		GenericHandlerAddr: genericHandlerAddress,
		AssetStoreAddr:     assetStoreAddress,
		GenericResourceID:  genericResourceID,

		Erc721Addr:        erc721ContractAddress,
		Erc721HandlerAddr: erc721HandlerContractAddress,
		Erc721ResourceID:  erc721ResourceID,

		IsBasicFeeHandler: true,
		Fee:               big.NewInt(100000000000),
		FeeHandlerAddr:    *feeHandlerContract.ContractAddress(),
	}

	err = SetupERC20Handler(bridgeContract, erc20Contract, mintTo, conf)
	if err != nil {
		return BridgeConfig{}, err
	}

	err = SetupERC721Handler(bridgeContract, erc721Contract, conf)
	if err != nil {
		return BridgeConfig{}, err
	}

	err = SetupGenericHandler(bridgeContract, conf)
	if err != nil {
		return BridgeConfig{}, err
	}

	err = SetupFeeHandler(bridgeContract, feeHandlerContract)
	if err != nil {
		return BridgeConfig{}, err
	}

	log.Debug().Msgf("All deployments and preparations are done")
	return conf, nil
}

func deployGeneric(
	ethClient EVMClient, t transactor.Transactor, bridgeContractAddress common.Address,
) (common.Address, common.Address, error) {
	assetStoreContract := centrifuge.NewAssetStoreContract(ethClient, common.Address{}, t)
	assetStoreAddress, err := assetStoreContract.DeployContract()
	if err != nil {
		return common.Address{}, common.Address{}, err
	}
	genericHandlerContract := generic.NewGenericHandlerContract(ethClient, common.Address{}, t)
	genericHandlerAddress, err := genericHandlerContract.DeployContract(bridgeContractAddress)
	if err != nil {
		return common.Address{}, common.Address{}, err
	}
	log.Debug().Msgf(
		"Centrifuge asset store deployed to: %s; \n Generic Handler deployed to: %s",
		assetStoreAddress, genericHandlerAddress,
	)
	return genericHandlerAddress, assetStoreAddress, nil
}

func deployErc20(
	ethClient EVMClient, t transactor.Transactor, bridgeContractAddress common.Address,
) (*erc20.ERC20Contract, common.Address, common.Address, error) {
	erc20Contract := erc20.NewERC20Contract(ethClient, common.Address{}, t)
	erc20ContractAddress, err := erc20Contract.DeployContract("Test", "TST")
	if err != nil {
		return nil, common.Address{}, common.Address{}, err
	}
	erc20HandlerContract := erc20.NewERC20HandlerContract(ethClient, common.Address{}, t)
	erc20HandlerContractAddress, err := erc20HandlerContract.DeployContract(bridgeContractAddress)
	if err != nil {
		return nil, common.Address{}, common.Address{}, err
	}
	log.Debug().Msgf(
		"Erc20 deployed to: %s; \n Erc20 Handler deployed to: %s",
		erc20ContractAddress, erc20HandlerContractAddress,
	)
	return erc20Contract, erc20ContractAddress, erc20HandlerContractAddress, nil
}

func deployErc721(
	ethClient EVMClient, t transactor.Transactor, bridgeContractAddress common.Address,
) (*erc721.ERC721Contract, common.Address, common.Address, error) {
	erc721Contract := erc721.NewErc721Contract(ethClient, common.Address{}, t)
	erc721ContractAddress, err := erc721Contract.DeployContract("TestERC721", "TST721", "")
	if err != nil {
		return nil, common.Address{}, common.Address{}, err
	}
	erc721HandlerContract := erc721.NewERC721HandlerContract(ethClient, common.Address{}, t)
	erc721HandlerContractAddress, err := erc721HandlerContract.DeployContract(bridgeContractAddress)
	if err != nil {
		return nil, common.Address{}, common.Address{}, err
	}
	log.Debug().Msgf(
		"Erc721 deployed to: %s; \n Erc721 Handler deployed to: %s",
		erc721ContractAddress, erc721HandlerContractAddress,
	)
	return erc721Contract, erc721ContractAddress, erc721HandlerContractAddress, nil
}

func deployFeeHandler(
	ethClient EVMClient, t transactor.Transactor, bridgeContractAddress common.Address,
) (*feeHandler.BasicFeeHandlerContract, error) {
	feeHandlerContract := feeHandler.NewBasicFeeHandlerContract(ethClient, common.Address{}, t)
	_, err := feeHandlerContract.DeployContract(bridgeContractAddress)
	if err != nil {
		return nil, err
	}

	return feeHandlerContract, nil
}

func SetupFeeHandler(bridgeContract *bridge.BridgeContract, feeHandlerContract *feeHandler.BasicFeeHandlerContract) error {
	_, err := bridgeContract.AdminChangeFeeHandler(*feeHandlerContract.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return err
	}

	basicFee := big.NewInt(100000000000)
	_, err = feeHandlerContract.ChangeFee(basicFee, transactor.TransactOptions{})
	if err != nil {
		return err
	}

	return nil
}

func SetupERC20Handler(
	bridgeContract *bridge.BridgeContract, erc20Contract *erc20.ERC20Contract, mintTo common.Address, conf BridgeConfig,
) error {
	_, err := bridgeContract.AdminSetResource(
		conf.Erc20HandlerAddr, conf.Erc20ResourceID, conf.Erc20Addr, transactor.TransactOptions{GasLimit: 2000000},
	)
	if err != nil {
		return err
	}
	// Minting tokens
	tenTokens := big.NewInt(0).Mul(big.NewInt(10), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))
	_, err = erc20Contract.MintTokens(mintTo, tenTokens, transactor.TransactOptions{})
	if err != nil {
		return err
	}
	// Approving tokens
	_, err = erc20Contract.ApproveTokens(conf.Erc20HandlerAddr, tenTokens, transactor.TransactOptions{})
	if err != nil {
		return err
	}
	// Adding minter
	_, err = erc20Contract.AddMinter(conf.Erc20HandlerAddr, transactor.TransactOptions{})
	if err != nil {
		return err
	}
	// Set burnable input
	_, err = bridgeContract.SetBurnableInput(conf.Erc20HandlerAddr, conf.Erc20Addr, transactor.TransactOptions{})
	if err != nil {
		return err
	}
	return nil
}

func SetupGenericHandler(bridgeContract *bridge.BridgeContract, conf BridgeConfig) error {
	_, err := bridgeContract.AdminSetGenericResource(
		conf.GenericHandlerAddr,
		conf.GenericResourceID,
		conf.AssetStoreAddr,
		[4]byte{0x65, 0x4c, 0xf8, 0x8c},
		big.NewInt(0),
		[4]byte{0x65, 0x4c, 0xf8, 0x8c},
		transactor.TransactOptions{GasLimit: 2000000},
	)
	if err != nil {
		return err
	}
	return nil
}

func SetupERC721Handler(bridgeContract *bridge.BridgeContract, erc721Contract *erc721.ERC721Contract, conf BridgeConfig) error {
	_, err := bridgeContract.AdminSetResource(conf.Erc721HandlerAddr, conf.Erc721ResourceID, conf.Erc721Addr, transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return err
	}
	// Adding minter
	_, err = erc721Contract.AddMinter(conf.Erc721HandlerAddr, transactor.TransactOptions{})
	if err != nil {
		return err
	}
	// Set burnable input
	_, err = bridgeContract.SetBurnableInput(conf.Erc721HandlerAddr, conf.Erc721Addr, transactor.TransactOptions{})
	if err != nil {
		return err
	}
	return nil
}
