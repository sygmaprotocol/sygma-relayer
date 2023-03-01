// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Events struct {
	types.EventRecords
	SygmaBridge_Deposit         []Deposit
	SygmaBasicFeeHandler_FeeSet []FeeSet

	SygmaBridge_ProposalExecution       []ProposalExecution
	SygmaBridge_FailedHandlerExecution  []FailedHandlerExecution
	SygmaBridge_Retry                   []Retry
	SygmaBridge_BridgePaused            []BridgePaused
	SygmaBridge_BridgeUnpaused          []BridgeUnpaused
	SygmaBridge_RegisterDestDomain      []RegisterDestDomain
	SygmaBridge_UnRegisterDestDomain    []UnregisterDestDomain
	SygmaFeeHandlerRouter_FeeHandlerSet []FeeHandlerSet
}

type Deposit struct {
	Phase        types.Phase
	DestDomainID types.U8
	ResourceID   types.Bytes32
	DepositNonce types.U64
	Sender       types.AccountID
	TransferType [1]byte
	CallData     []byte
	Handler      [1]byte
	Topics       []types.Hash
}

type FeeSet struct {
	Phase    types.Phase
	DomainID types.U8
	Asset    types.AssetID
	Amount   types.U128
	Topics   []types.Hash
}

type ProposalExecution struct {
	Phase          types.Phase
	OriginDomainID types.U8
	DepositNonce   types.U64
	DataHash       types.Bytes32
	Topics         []types.Hash
}

type FailedHandlerExecution struct {
	Phase          types.Phase
	Error          []byte
	OriginDomainID types.U8
	DepositNonce   types.U64
	Topics         []types.Hash
}

type Retry struct {
	Phase                types.Phase
	DepositOnBlockHeight types.U128
	DestDomainID         types.U8
	Sender               types.AccountID
	Topics               []types.Hash
}

type BridgePaused struct {
	Phase        types.Phase
	DestDomainID types.U8
	Topics       []types.Hash
}

type BridgeUnpaused struct {
	Phase        types.Phase
	DestDomainID types.U8
	Topics       []types.Hash
}

type RegisterDestDomain struct {
	Phase    types.Phase
	Sender   types.AccountID
	DomainID types.U8
	ChainID  types.U256
	Topics   []types.Hash
}

type UnregisterDestDomain struct {
	Phase    types.Phase
	Sender   types.AccountID
	DomainID types.U8
	ChainID  types.U256
	Topics   []types.Hash
}

type FeeHandlerSet struct {
	Phase       types.Phase
	DomainID    types.U8
	Asset       types.AssetID
	HandlerType [1]byte
	Topics      []types.Hash
}
