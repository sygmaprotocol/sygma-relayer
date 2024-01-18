// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"

	"github.com/ChainSafe/sygma-relayer/chains"
)

const (
	ERC20                 message.MessageType = "erc20"
	ERC721                message.MessageType = "erc721"
	PermissionedGeneric   message.MessageType = "permissionedGeneric"
	PermissionlessGeneric message.MessageType = "permissionlessGeneric"
)

type TransferMessageData struct {
	DepositNonce uint64
	ResourceId   [32]byte
	Payload      []interface{}
	Metadata     map[string]interface{}
}

type TransferMessage struct {
	Source      uint8
	Destination uint8
	Data        TransferMessageData
	Type        message.MessageType
}

type TransferMessageHandler struct{}

func (h *TransferMessageHandler) HandleMessage(msg *message.Message) (*proposal.Proposal, error) {

	transferMessage := &TransferMessage{
		Source:      msg.Source,
		Destination: msg.Destination,
		Data:        msg.Data.(TransferMessageData),
		Type:        msg.Type,
	}

	switch msg.Type {
	case ERC20:
		return ERC20MessageHandler(transferMessage)
	case ERC721:
		return ERC721MessageHandler(transferMessage)
	case PermissionedGeneric:
		return GenericMessageHandler(transferMessage)
	case PermissionlessGeneric:
		return PermissionlessGenericMessageHandler(transferMessage)
	}
	return nil, errors.New("wrong message type passed while handling message")
}

func PermissionlessGenericMessageHandler(msg *TransferMessage) (*proposal.Proposal, error) {

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
	return chains.NewProposal(msg.Source, msg.Destination, chains.TransferProposalData{
		DepositNonce: msg.Data.DepositNonce,
		ResourceId:   msg.Data.ResourceId,
		Metadata:     msg.Data.Metadata,
		Data:         data.Bytes(),
	}, chains.TransferProposalType), nil
}

func ERC20MessageHandler(msg *TransferMessage) (*proposal.Proposal, error) {
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
	var data []byte
	data = append(data, common.LeftPadBytes(amount, 32)...) // amount (uint256)
	recipientLen := big.NewInt(int64(len(recipient))).Bytes()
	data = append(data, common.LeftPadBytes(recipientLen, 32)...) // length of recipient (uint256)
	data = append(data, recipient...)                             // recipient ([]byte)

	return chains.NewProposal(msg.Source, msg.Destination, chains.TransferProposalData{
		DepositNonce: msg.Data.DepositNonce,
		ResourceId:   msg.Data.ResourceId,
		Metadata:     msg.Data.Metadata,
		Data:         data,
	}, chains.TransferProposalType), nil
}

func ERC721MessageHandler(msg *TransferMessage) (*proposal.Proposal, error) {

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
	return chains.NewProposal(msg.Source, msg.Destination, chains.TransferProposalData{
		DepositNonce: msg.Data.DepositNonce,
		ResourceId:   msg.Data.ResourceId,
		Metadata:     msg.Data.Metadata,
		Data:         data.Bytes(),
	}, chains.TransferProposalType), nil
}

func GenericMessageHandler(msg *TransferMessage) (*proposal.Proposal, error) {
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
	return chains.NewProposal(msg.Source, msg.Destination, chains.TransferProposalData{
		DepositNonce: msg.Data.DepositNonce,
		ResourceId:   msg.Data.ResourceId,
		Metadata:     msg.Data.Metadata,
		Data:         data.Bytes(),
	}, chains.TransferProposalType), nil
}
