// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"math/big"

	"github.com/ChainSafe/sygma-relayer/relayer/transfer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type DepositHandlerFunc func(sourceID uint8, calldata []byte) (*message.Message, error)

type BtcDepositHandler struct {
	depositHandler DepositHandlerFunc
}

// NewBtcDepositHandler creates an instance of BtcDepositHandler that contains
// handler functions for processing deposit events
func NewBtcDepositHandler() *BtcDepositHandler {
	return &BtcDepositHandler{}
}

func (e *BtcDepositHandler) HandleDeposit(sourceID uint8,
	destID uint8,
	depositNonce uint64,
	resourceID [32]byte,
	amount *big.Int,
	reciever common.Address,
	messageID string) (*message.Message, error) {

	evmAdd := reciever.Bytes()
	payload := []interface{}{
		amount.Bytes(),
		evmAdd,
	}

	return message.NewMessage(sourceID, destID, transfer.TransferMessageData{
		DepositNonce: depositNonce,
		ResourceId:   RESOURCE_ID,
		Metadata:     nil,
		Payload:      payload,
		Type:         transfer.FungibleTransfer,
	}, messageID, transfer.TransferMessageType), nil
}
