// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"

	"github.com/rs/zerolog/log"
)

type Handlers map[message.MessageType]MessageHandlerFunc
type MessageHandlerFunc func(m *message.Message) (*proposal.Proposal, error)

type SubstrateMessageHandler struct {
	handlers Handlers
}

// NewSubstrateMessageHandler creates an instance of SubstrateMessageHandler that contains
// message handler functions for converting deposit message into a chain specific
// proposal
func NewSubstrateMessageHandler() *SubstrateMessageHandler {
	return &SubstrateMessageHandler{
		handlers: make(map[message.MessageType]MessageHandlerFunc),
	}
}

func (mh *SubstrateMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	transferMessage := &chains.TransferMessage{
		Source:      m.Source,
		Destination: m.Destination,
		Data: chains.TransferMessageData{
			DepositNonce: m.Data.(chains.TransferMessageData).DepositNonce,
			ResourceId:   m.Data.(chains.TransferMessageData).ResourceId,
			Metadata:     m.Data.(chains.TransferMessageData).Metadata,
			Payload:      m.Data.(chains.TransferMessageData).Payload,
		},
		Type: m.Type,
	}

	// Based on handler that was registered on BridgeContract
	handleMessage, err := mh.matchTransferTypeHandlerFunc(m.Type)
	if err != nil {
		return nil, err
	}
	log.Info().Str("type", string(transferMessage.Type)).Uint8("src", transferMessage.Source).Uint8("dst", transferMessage.Destination).Uint64("nonce", transferMessage.Data.DepositNonce).Str("resourceID", fmt.Sprintf("%x", transferMessage.Data.ResourceId)).Msg("Handling new message")
	prop, err := handleMessage(m)
	if err != nil {
		return nil, err
	}
	return prop, nil
}

func (mh *SubstrateMessageHandler) matchTransferTypeHandlerFunc(transferType message.MessageType) (MessageHandlerFunc, error) {
	h, ok := mh.handlers[transferType]
	if !ok {
		return nil, fmt.Errorf("no corresponding message handler for this transfer type %s exists", transferType)
	}
	return h, nil
}

// RegisterEventHandler registers an message handler by associating a handler function to a specified transfer type
func (mh *SubstrateMessageHandler) RegisterMessageHandler(transferType message.MessageType, handler MessageHandlerFunc) {
	if transferType == "" {
		return
	}

	log.Info().Msgf("Registered message handler for transfer type %s", transferType)

	mh.handlers[transferType] = handler
}

func FungibleTransferMessageHandler(m *message.Message) (*proposal.Proposal, error) {
	transferMessage := &chains.TransferMessage{
		Source:      m.Source,
		Destination: m.Destination,
		Data: chains.TransferMessageData{
			DepositNonce: m.Data.(chains.TransferMessageData).DepositNonce,
			ResourceId:   m.Data.(chains.TransferMessageData).ResourceId,
			Metadata:     m.Data.(chains.TransferMessageData).Metadata,
			Payload:      m.Data.(chains.TransferMessageData).Payload,
		},
		Type: m.Type,
	}

	if len(transferMessage.Data.Payload) != 2 {
		return nil, errors.New("malformed payload. Len  of payload should be 2")
	}
	amount, ok := transferMessage.Data.Payload[0].([]byte)
	if !ok {
		return nil, errors.New("wrong payload amount format")
	}
	recipient, ok := transferMessage.Data.Payload[1].([]byte)
	if !ok {
		return nil, errors.New("wrong payload recipient format")
	}
	var data []byte
	data = append(data, common.LeftPadBytes(amount, 32)...) // amount (uint256)

	recipientLen := big.NewInt(int64(len(recipient))).Bytes()
	data = append(data, common.LeftPadBytes(recipientLen, 32)...)
	data = append(data, recipient...)
	return chains.NewTransferProposal(transferMessage.Source, transferMessage.Destination, transferMessage.Data.DepositNonce, transferMessage.Data.ResourceId, transferMessage.Data.Metadata, data, chains.TransferProposalType), nil
}
