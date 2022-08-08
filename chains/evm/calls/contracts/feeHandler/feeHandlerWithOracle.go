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

type FeeHandlerWithOracleContract struct {
	contracts.Contract
}

func NewFeeHandlerWithOracleContract(
	client calls.ContractCallerDispatcher,
	feeHandlerWithOracleContractAddress common.Address,
	transactor transactor.Transactor,
) *FeeHandlerWithOracleContract {
	a, _ := abi.JSON(strings.NewReader(consts.FeeHandlerWithOracleABI))
	b := common.FromHex(consts.FeeHandlerWithOracleBin)
	return &FeeHandlerWithOracleContract{contracts.NewContract(feeHandlerWithOracleContractAddress, a, b, client, transactor)}
}

func (f *FeeHandlerWithOracleContract) SetFeeOracle(
	oracleAddr common.Address,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().Msgf("setting fee oracle address %s", oracleAddr.String())
	return f.ExecuteTransaction(
		"setFeeOracle",
		opts,
		oracleAddr,
	)
}

func (f *FeeHandlerWithOracleContract) SetFeeProperties(
	gasUsed uint32,
	feePercent uint16,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().Msgf("setting fee properties: gasUsed: %v, feePercent: %v", gasUsed, float32(feePercent)/float32(100))
	return f.ExecuteTransaction(
		"setFeeProperties",
		opts,
		gasUsed,
		feePercent,
	)
}

func (f *FeeHandlerWithOracleContract) DistributeFee(
	resourceID types.ResourceID,
	addrs []common.Address,
	amounts []*big.Int,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().Msgf("distributing the fee to: addresses: %v, amounts: %v, resourceId: %v", addrs, amounts, resourceID)
	return f.ExecuteTransaction(
		"transferFee",
		opts,
		resourceID,
		addrs,
		amounts,
	)
}

func (f *FeeHandlerWithOracleContract) CalculateFee(
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
