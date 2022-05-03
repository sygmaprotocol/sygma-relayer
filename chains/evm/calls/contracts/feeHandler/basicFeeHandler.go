package feeHandler

import (
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/consts"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"strings"
)

type BasicFeeHandlerContract struct {
	contracts.Contract
}

func NewBasicFeeHandlerContract(
	client calls.ContractCallerDispatcher,
	basicFeeHandlerContractAddress common.Address,
	transactor transactor.Transactor,
) *BasicFeeHandlerContract {
	a, _ := abi.JSON(strings.NewReader(consts.BasicFeeHandlerABI))
	b := common.FromHex(consts.BasicFeeHandlerBin)
	return &BasicFeeHandlerContract{contracts.NewContract(basicFeeHandlerContractAddress, a, b, client, transactor)}
}

func (b *BasicFeeHandlerContract) ChangeFee(
	newFee uint64,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().Msgf("Changing new fee %v", newFee)
	return b.ExecuteTransaction(
		"changeFee",
		opts,
		newFee,
	)
}
