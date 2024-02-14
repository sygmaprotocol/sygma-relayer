// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"errors"
	"math/big"

	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ChainSafe/sygma-relayer/relayer/transfer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type SubstrateMessageHandler struct{}

func (mh *SubstrateMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	transferMessage := &transfer.TransferMessage{
		Source:      m.Source,
		Destination: m.Destination,
		Data: transfer.TransferMessageData{
			DepositNonce: m.Data.(transfer.TransferMessageData).DepositNonce,
			ResourceId:   m.Data.(transfer.TransferMessageData).ResourceId,
			Metadata:     m.Data.(transfer.TransferMessageData).Metadata,
			Payload:      m.Data.(transfer.TransferMessageData).Payload,
			Type:         m.Data.(transfer.TransferMessageData).Type,
		},
		Type: m.Type,
	}
	switch transferMessage.Data.Type {
	case transfer.ERC20Message:
		return fungibleTransferMessageHandler(transferMessage)
	}
	return nil, errors.New("wrong message type passed while handling message")
}

func fungibleTransferMessageHandler(m *transfer.TransferMessage) (*proposal.Proposal, error) {

	if len(m.Data.Payload) != 2 {
		return nil, errors.New("malformed payload. Len  of payload should be 2")
	}
	amount, ok := m.Data.Payload[0].([]byte)
	if !ok {
		return nil, errors.New("wrong payload amount format")
	}
	recipient, ok := m.Data.Payload[1].([]byte)
	if !ok {
		return nil, errors.New("wrong payload recipient format")
	}
	var data []byte
	data = append(data, common.LeftPadBytes(amount, 32)...) // amount (uint256)

	recipientLen := big.NewInt(int64(len(recipient))).Bytes()
	data = append(data, common.LeftPadBytes(recipientLen, 32)...)
	data = append(data, recipient...)
	return chains.NewProposal(m.Source, m.Destination, transfer.TransferProposalData{
		DepositNonce: m.Data.DepositNonce,
		ResourceId:   m.Data.ResourceId,
		Metadata:     m.Data.Metadata,
		Data:         data,
	}, transfer.TransferProposalType), nil
}
