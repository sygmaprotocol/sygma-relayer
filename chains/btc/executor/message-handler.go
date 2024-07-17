// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"errors"
	"math/big"

	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"

	"github.com/ChainSafe/sygma-relayer/relayer/transfer"
)

type BtcTransferProposalData struct {
	Amount       uint64
	Recipient    string
	DepositNonce uint64
	ResourceId   [32]byte
}

type BtcTransferProposal struct {
	Source      uint8
	Destination uint8
	Data        BtcTransferProposalData
}

type BtcMessageHandler struct{}

func (h *BtcMessageHandler) HandleMessage(msg *message.Message) (*proposal.Proposal, error) {
	transferMessage := &transfer.TransferMessage{
		Source:      msg.Source,
		Destination: msg.Destination,
		Data:        msg.Data.(transfer.TransferMessageData),
		Type:        msg.Type,
		ID:          msg.ID,
	}

	switch transferMessage.Data.Type {
	case transfer.FungibleTransfer:
		return ERC20MessageHandler(transferMessage)
	}
	return nil, errors.New("wrong message type passed while handling message")
}

func ERC20MessageHandler(msg *transfer.TransferMessage) (*proposal.Proposal, error) {
	if len(msg.Data.Payload) != 2 {
		return nil, errors.New("malformed payload. Len  of payload should be 2")
	}
	amount, ok := msg.Data.Payload[0].([]byte)
	if !ok {
		return nil, errors.New("wrong payload amount format")
	}
	recipient, ok := msg.Data.Payload[1].([]byte)
	if !ok {
		return nil, errors.New("wrong payload recipient format")
	}
	bigAmount := new(big.Int).SetBytes(amount)

	// remove 10 decimal places to match Bitcoin network
	divisor := new(big.Int)
	divisor.Exp(big.NewInt(10), big.NewInt(10), nil)
	bigAmount.Div(bigAmount, divisor)

	return proposal.NewProposal(msg.Source, msg.Destination, BtcTransferProposalData{
		Amount:       bigAmount.Uint64(),
		Recipient:    string(recipient),
		DepositNonce: msg.Data.DepositNonce,
		ResourceId:   msg.Data.ResourceId,
	}, msg.ID, transfer.TransferProposalType), nil
}
