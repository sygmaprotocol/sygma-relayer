// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Deposit struct {
	DestDomainID types.U8      `mapstructure:"dest_domain_id"`
	ResourceID   types.Bytes32 `mapstructure:"resource_id"`
	DepositNonce types.U64     `mapstructure:"deposit_nonce"`
	TransferType types.U8      `mapstructure:"sygma_traits_TransferType"`
	CallData     []byte        `mapstructure:"deposit_data"`
	Handler      [1]byte       `mapstructure:"handler_response"`
}

type Retry struct {
	DepositOnBlockHeight types.U128 `mapstructure:"deposit_on_block_height"`
	DestDomainID         types.U8   `mapstructure:"dest_domain_id"`
}

const (
	CodeUpdatedEvent      = "System.CodeUpdated"
	ExtrinsicFailedEvent  = "System.ExtrinsicFailed"
	ExtrinsicSuccessEvent = "System.ExtrinsicSuccess"
	RetryEvent            = "SygmaBridge.Retry"
	DepositEvent          = "SygmaBridge.Deposit"
)
