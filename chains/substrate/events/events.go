// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	core_types "github.com/ChainSafe/chainbridge-core/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Deposit struct {
	DestinationDomainID uint8
	ResourceID          core_types.ResourceID
	DepositNonce        uint64
	DepositType         message.TransferType
	SenderAddress       types.AccountID
	Data                []byte
	HandlerResponse     []byte
}

type Events struct {
	types.EventRecords
	Deposit []Deposit
}
