// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package retry

import (
	"math/big"

	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

const (
	RetryMessageType message.MessageType = "RetryMessage"
)

type RetryMessageData struct {
	SourceDomainID      uint8
	DestinationDomainID uint8
	BlockHeight         *big.Int
}

type RetryMessage struct {
	Source      uint8
	Destination uint8
	Data        RetryMessageData
	Type        message.MessageType
	ID          string
}
