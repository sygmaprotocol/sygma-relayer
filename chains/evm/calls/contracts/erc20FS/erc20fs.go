package erc20FS

import (
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-hub/chains/evm/calls/consts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"math/big"
	"strings"
)

type Erc20FixedSupplyContract struct {
	contracts.Contract
}

func NewErc20FixedSupplyContract(
	client calls.ContractCallerDispatcher,
	bridgeContractAddress common.Address,
	transactor transactor.Transactor,
) *Erc20FixedSupplyContract {
	a, _ := abi.JSON(strings.NewReader(consts.Erc20LRABI))
	b := common.FromHex(consts.Erc20LRBin)
	return &Erc20FixedSupplyContract{contracts.NewContract(bridgeContractAddress, a, b, client, transactor)}
}

func (c *Erc20FixedSupplyContract) Transfer(
	to common.Address,
	amount *big.Int,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().Msgf("Transferring %s tokens to %s", amount.String(), to.String())
	return c.ExecuteTransaction("transfer", opts, to, amount)
}
