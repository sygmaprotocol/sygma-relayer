// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"

	"github.com/ChainSafe/sygma-relayer/chains/evm/listener/depositHandlers"

	"github.com/ChainSafe/sygma-relayer/relayer/transfer"
)

type TransferMessageHandler struct{}

func (h *TransferMessageHandler) HandleMessage(msg *message.Message) (*proposal.Proposal, error) {
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
	case transfer.SemiFungibleTransfer:
		return ERC1155MessageHandler(transferMessage)
	case transfer.NonFungibleTransfer:
		return ERC721MessageHandler(transferMessage)
	case transfer.PermissionedGenericTransfer:
		return GenericMessageHandler(transferMessage)
	case transfer.PermissionlessGenericTransfer:
		return PermissionlessGenericMessageHandler(transferMessage)
	}
	return nil, errors.New("wrong message type passed while handling message")
}

func PermissionlessGenericMessageHandler(msg *transfer.TransferMessage) (*proposal.Proposal, error) {
	executeFunctionSignature, ok := msg.Data.Payload[0].([]byte)
	if !ok {
		return nil, errors.New("wrong function signature format")
	}
	executeContractAddress, ok := msg.Data.Payload[1].([]byte)
	if !ok {
		return nil, errors.New("wrong contract address format")
	}
	maxFee, ok := msg.Data.Payload[2].([]byte)
	if !ok {
		return nil, errors.New("wrong max fee format")
	}
	depositor, ok := msg.Data.Payload[3].([]byte)
	if !ok {
		return nil, errors.New("wrong depositor data format")
	}
	executionData, ok := msg.Data.Payload[4].([]byte)
	if !ok {
		return nil, errors.New("wrong execution data format")
	}

	data := bytes.Buffer{}
	data.Write(common.LeftPadBytes(maxFee, 32))

	data.Write(common.LeftPadBytes(big.NewInt(int64(len(executeFunctionSignature))).Bytes(), 2))
	data.Write(executeFunctionSignature)

	data.Write([]byte{byte(len(executeContractAddress))})
	data.Write(executeContractAddress)

	data.Write([]byte{byte(len(depositor))})
	data.Write(depositor)

	data.Write(executionData)
	return proposal.NewProposal(msg.Source, msg.Destination, transfer.TransferProposalData{
		DepositNonce: msg.Data.DepositNonce,
		ResourceId:   msg.Data.ResourceId,
		Metadata:     msg.Data.Metadata,
		Data:         data.Bytes(),
	}, msg.ID, transfer.TransferProposalType), nil
}

func ERC20MessageHandler(msg *transfer.TransferMessage) (*proposal.Proposal, error) {
	if len(msg.Data.Payload) != 2 && len(msg.Data.Payload) != 3 {
		return nil, fmt.Errorf("wrong payload length %d", len(msg.Data.Payload))
	}
	amount, ok := msg.Data.Payload[0].([]byte)
	if !ok {
		return nil, errors.New("wrong payload amount format")
	}
	recipient, ok := msg.Data.Payload[1].([]byte)
	if !ok {
		return nil, errors.New("wrong payload recipient format")
	}
	var data []byte
	data = append(data, common.LeftPadBytes(amount, 32)...) // amount (uint256)
	recipientLen := big.NewInt(int64(len(recipient))).Bytes()
	data = append(data, common.LeftPadBytes(recipientLen, 32)...) // length of recipient (uint256)
	data = append(data, recipient...)                             // recipient ([]byte)
	if len(msg.Data.Payload) == 3 {
		optionalMessage, ok := msg.Data.Payload[2].([]byte)
		if !ok {
			return nil, errors.New("wrong optional message format")
		}

		data = append(data, optionalMessage...)
	}

	return proposal.NewProposal(msg.Source, msg.Destination, transfer.TransferProposalData{
		DepositNonce: msg.Data.DepositNonce,
		ResourceId:   msg.Data.ResourceId,
		Metadata:     msg.Data.Metadata,
		Data:         data,
	}, msg.ID, transfer.TransferProposalType), nil
}

func ERC721MessageHandler(msg *transfer.TransferMessage) (*proposal.Proposal, error) {
	if len(msg.Data.Payload) != 3 {
		return nil, errors.New("malformed payload. Len  of payload should be 3")
	}
	tokenID, ok := msg.Data.Payload[0].([]byte)
	if !ok {
		return nil, errors.New("wrong payload tokenID format")
	}
	recipient, ok := msg.Data.Payload[1].([]byte)
	if !ok {
		return nil, errors.New("wrong payload recipient format")
	}
	metadata, ok := msg.Data.Payload[2].([]byte)
	if !ok {
		return nil, errors.New("wrong payload metadata format")
	}
	data := bytes.Buffer{}
	data.Write(common.LeftPadBytes(tokenID, 32))
	recipientLen := big.NewInt(int64(len(recipient))).Bytes()
	data.Write(common.LeftPadBytes(recipientLen, 32))
	data.Write(recipient)
	metadataLen := big.NewInt(int64(len(metadata))).Bytes()
	data.Write(common.LeftPadBytes(metadataLen, 32))
	data.Write(metadata)
	return proposal.NewProposal(msg.Source, msg.Destination, transfer.TransferProposalData{
		DepositNonce: msg.Data.DepositNonce,
		ResourceId:   msg.Data.ResourceId,
		Metadata:     msg.Data.Metadata,
		Data:         data.Bytes(),
	}, msg.ID, transfer.TransferProposalType), nil
}

func ERC1155MessageHandler(msg *transfer.TransferMessage) (*proposal.Proposal, error) {
	if len(msg.Data.Payload) != 4 {
		return nil, errors.New("malformed payload. Len  of payload should be 4")
	}
	_, ok := msg.Data.Payload[0].([]*big.Int)
	if !ok {
		return nil, errors.New("wrong payload tokenID format")
	}
	_, ok = msg.Data.Payload[1].([]*big.Int)
	if !ok {
		return nil, errors.New("wrong payload amount format")
	}
	_, ok = msg.Data.Payload[2].([]byte)
	if !ok {
		return nil, errors.New("wrong payload recipient format")
	}
	if len(msg.Data.Payload[2].([]byte)) != 20 {
		return nil, errors.New("malformed payload. Len  of recipient should be 20")
	}
	_, ok = msg.Data.Payload[3].([]byte)
	if !ok {
		return nil, errors.New("wrong payload transferData format")
	}

	erc1155Type, err := depositHandlers.GetErc1155Type()
	if err != nil {
		return nil, err
	}

	data, err := erc1155Type.PackValues(msg.Data.Payload)
	if err != nil {
		return nil, err
	}

	return proposal.NewProposal(msg.Source, msg.Destination, transfer.TransferProposalData{
		DepositNonce: msg.Data.DepositNonce,
		ResourceId:   msg.Data.ResourceId,
		Metadata:     msg.Data.Metadata,
		Data:         data,
	}, msg.ID, transfer.TransferProposalType), nil
}

func GenericMessageHandler(msg *transfer.TransferMessage) (*proposal.Proposal, error) {
	if len(msg.Data.Payload) != 1 {
		return nil, errors.New("malformed payload. Len  of payload should be 1")
	}
	metadata, ok := msg.Data.Payload[0].([]byte)
	if !ok {
		return nil, errors.New("wrong payload metadata format")
	}
	data := bytes.Buffer{}
	metadataLen := big.NewInt(int64(len(metadata))).Bytes()
	data.Write(common.LeftPadBytes(metadataLen, 32)) // length of metadata (uint256)
	data.Write(metadata)
	return proposal.NewProposal(msg.Source, msg.Destination, transfer.TransferProposalData{
		DepositNonce: msg.Data.DepositNonce,
		ResourceId:   msg.Data.ResourceId,
		Metadata:     msg.Data.Metadata,
		Data:         data.Bytes(),
	}, msg.ID, transfer.TransferProposalType), nil
}
