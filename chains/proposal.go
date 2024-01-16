// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package chains

import (
	"fmt"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/types"
	coreMessage "github.com/sygmaprotocol/sygma-core/relayer/message"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

const (
	TransferProposalType proposal.ProposalType   = "Transfer"
	TransferMessageType  coreMessage.MessageType = "Transfer"
)

type TransferProposal struct {
	Source      uint8
	Destination uint8
	Data        TransferProposalData
	Type        proposal.ProposalType
}

type TransferProposalData struct {
	DepositNonce uint64
	ResourceId   [32]byte
	Metadata     map[string]interface{}
	Data         []byte
}

func NewTransferProposal(source, destination uint8, depositNonce uint64,
	resourceId [32]byte, metadata map[string]interface{}, data []byte, propType proposal.ProposalType) *TransferProposal {

	transferProposalData := TransferProposalData{
		DepositNonce: depositNonce,
		ResourceId:   resourceId,
		Metadata:     metadata,
		Data:         data,
	}

	return &TransferProposal{
		Source:      source,
		Destination: destination,
		Data:        transferProposalData,
		Type:        propType,
	}
}

func NewProposal(source uint8, destination uint8, data interface{}, propType proposal.ProposalType) *proposal.Proposal {
	return &proposal.Proposal{
		Source:      source,
		Destination: destination,
		Data:        data,
		Type:        propType,
	}
}

type Proposal struct {
	OriginDomainID uint8            // Source domainID where message was initiated
	DepositNonce   uint64           // Nonce for the deposit
	ResourceID     types.ResourceID // change id -> ID
	Data           []byte
	Destination    uint8 // Destination domainID where message is to be sent
	Metadata       message.Metadata
}

func ProposalsHash(proposals []*TransferProposal, chainID int64, verifContract string, bridgeVersion string) ([]byte, error) {

	formattedProps := make([]interface{}, len(proposals))
	for i, prop := range proposals {
		transferProposal := &TransferProposal{
			Source:      prop.Source,
			Destination: prop.Destination,
			Data: TransferProposalData{
				DepositNonce: prop.Data.DepositNonce,
				ResourceId:   prop.Data.ResourceId,
				Metadata:     prop.Data.Metadata,
				Data:         prop.Data.Data,
			},
			Type: prop.Type,
		}
		formattedProps[i] = map[string]interface{}{
			"originDomainID": math.NewHexOrDecimal256(int64(transferProposal.Source)),
			"depositNonce":   math.NewHexOrDecimal256(int64(transferProposal.Data.DepositNonce)),
			"resourceID":     hexutil.Encode(transferProposal.Data.ResourceId[:]),
			"data":           prop.Data,
		}
	}
	message := apitypes.TypedDataMessage{
		"proposals": formattedProps,
	}
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Proposal": []apitypes.Type{
				{Name: "originDomainID", Type: "uint8"},
				{Name: "depositNonce", Type: "uint64"},
				{Name: "resourceID", Type: "bytes32"},
				{Name: "data", Type: "bytes"},
			},
			"Proposals": []apitypes.Type{
				{Name: "proposals", Type: "Proposal[]"},
			},
		},
		PrimaryType: "Proposals",
		Domain: apitypes.TypedDataDomain{
			Name:              "Bridge",
			ChainId:           math.NewHexOrDecimal256(chainID),
			Version:           bridgeVersion,
			VerifyingContract: verifContract,
		},
		Message: message,
	}

	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return []byte{}, err
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return []byte{}, err
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	return crypto.Keccak256(rawData), nil
}

type TransferMessageData struct {
	DepositNonce uint64
	ResourceId   [32]byte
	Metadata     map[string]interface{}
	Payload      []interface{}
}

func NewTransferMessage(source, destination uint8, depositNonce uint64,
	resourceId [32]byte, metadata map[string]interface{}, payload []interface{}, msgType coreMessage.MessageType) *coreMessage.Message {

	transferMessage := TransferMessageData{
		DepositNonce: depositNonce,
		ResourceId:   resourceId,
		Metadata:     metadata,
		Payload:      payload,
	}

	return &coreMessage.Message{
		Source:      source,
		Destination: destination,
		Data:        transferMessage,
		Type:        msgType,
	}
}
