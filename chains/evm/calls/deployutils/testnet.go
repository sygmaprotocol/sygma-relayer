// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package deployutils

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmgaspricer"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/signAndSend"
	"github.com/ChainSafe/chainbridge-core/chains/evm/cli/flags"
	"github.com/ChainSafe/chainbridge-core/types"
	"github.com/ethereum/go-ethereum/common"
)

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

// SetupSygmaTestnet deploys all neccessary smart contracts that in current time deployed on TestNet environment. Should be used for test purposes
func SetupSygmaTestnet(dc *DeployConfig) (*DeployResults, error) {

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

	erc20Contract, err := DeployERC20Token(ethClient, t, dc.Erc20Name, dc.Erc20Symbol)
	if err != nil {
		return nil, err
	}
	// Setting up resourceID for ERC20 token
	_, err = bridgeContract.AdminSetResource(*erc20HandlerContract.ContractAddress(), rID, *erc20Contract.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}

	// Deploy FeeRouter and set it up on the bridge contact
	fr, err := SetupFeeRouter(ethClient, t, bridgeContract)
	if err != nil {
		return nil, err
	}

	// Deploy FeeHandlerWithOracle set FeeOracle address and FeeProperties
	fh, err := SetupFeeHandlerWithOracle(ethClient, t, *bridgeContract.ContractAddress(), *fr.ContractAddress(), dc.FeeOracleAddress, dc.FeeGas, dc.FeePercent)
	if err != nil {
		return nil, err
	}

	// Set provided resourceID on FeeRouter
	_, err = fr.AdminSetResourceHandler(dc.DestDomainID, rID, *fh.ContractAddress(), transactor.TransactOptions{GasLimit: 2000000})
	if err != nil {
		return nil, err
	}

	dr := &DeployResults{
		BridgeAddr:           *bridgeContract.ContractAddress(),
		DomainID:             dc.DomainID,
		AccessControlAddress: *accessControlContract.ContractAddress(),
		FeeHandlerAddress:    *fh.ContractAddress(),
		FeeRouterAddress:     *fr.ContractAddress(),
		Erc20HandlerAddr:     *erc20HandlerContract.ContractAddress(),
		Erc20Addr:            *erc20Contract.ContractAddress(),
		ERC20Sybmol:          dc.Erc20Symbol,
		Erc20ResourceID:      rID,
		Fee:                  dc.FeePercent,
	}

	return dr, nil
}
