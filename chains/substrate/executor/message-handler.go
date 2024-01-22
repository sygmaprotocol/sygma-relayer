// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"errors"
	"math/big"

	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type SubstrateMessageHandler struct{}

func (mh *SubstrateMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	transferMessage := &chains.TransferMessage{
		Source:      m.Source,
		Destination: m.Destination,
		Data:        m.Data.(chains.TransferMessageData),
		Type:        m.Type,
	}
	switch transferMessage.Type {
	case FungibleTransfer:
		return fungibleTransferMessageHandler(transferMessage)
	}
	return nil, errors.New("wrong message type passed while handling message")
}

func fungibleTransferMessageHandler(m *chains.TransferMessage) (*proposal.Proposal, error) {

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
	return chains.NewProposal(m.Source, m.Destination, chains.TransferProposalData{
		DepositNonce: m.Data.DepositNonce,
		ResourceId:   m.Data.ResourceId,
		Metadata:     m.Data.Metadata,
		Data:         data,
	}, chains.TransferProposalType), nil
}
