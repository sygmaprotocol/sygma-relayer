package feeHandler

import (
	"strings"

	"github.com/ChainSafe/sygma-core/chains/evm/calls"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/contracts"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/sygma-core/types"
	"github.com/ChainSafe/sygma/chains/evm/calls/consts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type FeeRouter struct {
	contracts.Contract
}

func NewFeeRouter(
	client calls.ContractCallerDispatcher,
	feeRouterAddress common.Address,
	transactor transactor.Transactor,
) *FeeRouter {
	a, _ := abi.JSON(strings.NewReader(consts.FeeRouterABI))
	b := common.FromHex(consts.FeeRouterBin)
	return &FeeRouter{contracts.NewContract(feeRouterAddress, a, b, client, transactor)}
}

// AdminSetResourceHandler sets handler for provided domainID and resourceID. https://github.com/ChainSafe/sygma-solidity/blob/master/contracts/handlers/FeeHandlerRouter.sol#L54
func (c *FeeRouter) AdminSetResourceHandler(destDomainID uint8, resourceID types.ResourceID, feeHandlerAddress common.Address, opts transactor.TransactOptions) (*common.Hash, error) {
	return c.ExecuteTransaction(
		"adminSetResourceHandler",
		opts,
		destDomainID,
		resourceID,
		feeHandlerAddress,
	)
}
