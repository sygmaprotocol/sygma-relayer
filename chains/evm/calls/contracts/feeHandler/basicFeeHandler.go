// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package feeHandler

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/types"

	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/consts"
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
	newFee *big.Int,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().Msgf("Changing new fee %v", newFee)
	return b.ExecuteTransaction(
		"changeFee",
		opts,
		newFee,
	)
}

func (b *BasicFeeHandlerContract) DistributeFee(
	addrs []common.Address,
	amounts []*big.Int,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().Msgf("distributing the fee to: addresses: %v, amounts: %v", addrs, amounts)
	return b.ExecuteTransaction(
		"transferFee",
		opts,
		addrs,
		amounts,
	)
}

func (f *BasicFeeHandlerContract) CalculateFee(
	sender common.Address,
	fromDomainID uint8,
	destDomainID uint8,
	resourceID types.ResourceID,
	data []byte,
	feeData []byte,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().Msgf("Calculating fee for deposit from sender: %s, from domain: %v, to domain: %v, for resourceID: %v", sender, fromDomainID, destDomainID, hexutil.Encode(resourceID[:]))
	return f.ExecuteTransaction(
		"calculateFee",
		opts,
		sender,
		fromDomainID,
		destDomainID,
		resourceID,
		data,
		feeData,
	)
}
