// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Events struct {
	types.EventRecords
	SygmaBridge_Deposit         []EventDeposit
	SygmaBasicFeeHandler_FeeSet []EventFeeSet

	SygmaBridge_ProposalExecution      []EventProposalExecution
	SygmaBridge_FailedHandlerExecution []EventFailedHandlerExecution
	SygmaBridge_Retry                  []EventRetry
	SygmaBridge_BridgePaused           []EventBridgePaused
	SygmaBridge_BridgeUnpaused         []EventBridgeUnpaused
}
type EventDeposit struct {
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

type EventFeeSet struct {
	Phase    types.Phase
	DomainID types.U8
	Asset    types.U32
	Amount   types.U64
	Topics   []types.Hash
}

type EventProposalExecution struct {
	Phase          types.Phase
	OriginDomainID types.U8
	DepositNonce   types.U64
	DataHash       types.Bytes32
	Topics         []types.Hash
}

type EventFailedHandlerExecution struct {
	Phase          types.Phase
	Error          []byte
	OriginDomainID types.U8
	DepositNonce   types.U64
	Topics         []types.Hash
}

type EventRetry struct {
	Phase                types.Phase
	DepositOnBlockHeight types.U128
	DestDomainID         types.U8
	Topics               []types.Hash
}

type EventBridgePaused struct {
	Phase        types.Phase
	DestDomainID types.U8
	Topics       []types.Hash
}

type EventBridgeUnpaused struct {
	Phase        types.Phase
	DestDomainID types.U8
	Topics       []types.Hash
}
