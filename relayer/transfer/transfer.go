package transfer

import (
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type TransferType string

const (
	FungibleTransfer              TransferType = "fungible"
	SemiFungibleTransfer          TransferType = "semiFungible"
	NonFungibleTransfer           TransferType = "nonFungible"
	PermissionedGenericTransfer   TransferType = "permissionedGeneric"
	PermissionlessGenericTransfer TransferType = "permissionlessGeneric"
)

type TransferMessageData struct {
	DepositNonce uint64
	ResourceId   [32]byte
	Metadata     map[string]interface{}
	Payload      []interface{}
	Type         TransferType
}

const (
	TransferMessageType  message.MessageType   = "TransferMessage"
	TransferProposalType proposal.ProposalType = "TransferProposal"
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
