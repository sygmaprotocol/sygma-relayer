// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package deployutils

import (
	"encoding/hex"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/centrifuge"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/erc20"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/erc721"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/keystore"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/accessControlSegregator"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/erc20Handler"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/generic"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

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
	"d8236744", // refreshKey
	"a973ec93", //grantAccess
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

func DeployERC20Token(ethClient EVMClient, t transactor.Transactor, Erc20Name, Erc20Symbol string) (*erc20.ERC20Contract, error) {
	erc20Contract := erc20.NewERC20Contract(ethClient, common.Address{}, t)
	_, err := erc20Contract.DeployContract(Erc20Name, Erc20Symbol)
	if err != nil {
		return nil, err
	}
	return erc20Contract, nil
}

func DeployErc20Handler(ethClient EVMClient, t transactor.Transactor, bridgeContractAddress common.Address) (*erc20Handler.ERC20HandlerContract, error) {
	erc20HandlerContract := erc20Handler.NewERC20HandlerContract(ethClient, common.Address{}, t)
	_, err := erc20HandlerContract.DeployContract(bridgeContractAddress)
	if err != nil {
		return nil, err
	}
	return erc20HandlerContract, nil
}

func SetupERC20Handler(
	bridgeContract *bridge.BridgeContract, erc20Contract *erc20.ERC20Contract, mintTo common.Address, conf *BridgeConfig,
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

func SetupERC20LockReleaseHandler(
	bridgeContract *bridge.BridgeContract, erc20Contract *erc20.ERC20Contract, mintTo common.Address, conf *BridgeConfig,
) error {
	_, err := bridgeContract.AdminSetResource(
		conf.Erc20LockReleaseHandlerAddr, conf.Erc20LockReleaseResourceID, conf.Erc20LockReleaseAddr, transactor.TransactOptions{GasLimit: 2000000},
	)
	if err != nil {
		return err
	}

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

	return nil
}

func SetupGenericHandler(bridgeContract *bridge.BridgeContract, conf *BridgeConfig) error {
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

func SetupERC721Handler(bridgeContract *bridge.BridgeContract, erc721Contract *erc721.ERC721Contract, conf *BridgeConfig) error {
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

var AliceKp = keystore.TestKeyRing.EthereumKeys[keystore.AliceKey]
var BobKp = keystore.TestKeyRing.EthereumKeys[keystore.BobKey]
var EveKp = keystore.TestKeyRing.EthereumKeys[keystore.EveKey]
var CharlieKp = keystore.TestKeyRing.EthereumKeys[keystore.CharlieKey]

var (
	MpcAddress = common.HexToAddress("0x1c5541A79AcC662ab2D2647F3B141a3B7Cdb2Ae4")
)

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

func deployErc20LockRelease(
	ethClient EVMClient, t transactor.Transactor, bridgeContractAddress common.Address,
) (*erc20.ERC20Contract, common.Address, error) {
	contract := erc20.NewERC20Contract(ethClient, common.Address{}, t)
	contractAddress, err := contract.DeployContract(
		"TestLockRelease", "TLR",
	)
	if err != nil {
		return nil, common.Address{}, err
	}
	log.Debug().Msgf(
		"Erc20LockRelease deployed to: %s",
		contractAddress,
	)
	return contract, contractAddress, nil
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
