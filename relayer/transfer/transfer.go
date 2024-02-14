package transfer

import (
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type TransferType string

const (
	ERC20Transfer                 TransferType = "erc20"
	ERC721Transfer                TransferType = "erc721"
	PermissionedGenericTransfer   TransferType = "permissionedGeneric"
	PermissionlessGenericTransfer TransferType = "permissionlessGeneric"
)

type TransferMessageDataType string

const (
	ERC20Message                 TransferMessageDataType = "erc20"
	ERC721Message                TransferMessageDataType = "erc721"
	PermissionedGenericMessage   TransferMessageDataType = "permissionedGeneric"
	PermissionlessGenericMessage TransferMessageDataType = "permissionlessGeneric"
)

type TransferMessageData struct {
	DepositNonce uint64
	ResourceId   [32]byte
	Metadata     map[string]interface{}
	Payload      []interface{}
	Type         TransferMessageDataType
}

const (
	TransferMessageType  message.MessageType   = "Transfer"
	TransferProposalType proposal.ProposalType = "Transfer"
)

type TransferMessage struct {
	Source      uint8
	Destination uint8
	Data        TransferMessageData
	Type        message.MessageType
}

type TransferProposalData struct {
	DepositNonce uint64
	ResourceId   [32]byte
	Metadata     map[string]interface{}
	Data         []byte
}

type TransferProposal struct {
	Source      uint8
	Destination uint8
	Data        TransferProposalData
	Type        proposal.ProposalType
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
