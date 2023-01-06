package proposal

import (
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/types"
)

func NewProposal(source, destination uint8, depositNonce uint64, resourceId types.ResourceID, data []byte) *Proposal {

	return &Proposal{
		Source:       source,
		Destination:  destination,
		DepositNonce: depositNonce,
		ResourceId:   resourceId,
		Data:         data,
	}
}

type Proposal struct {
	Source       uint8  // Source domainID where message was initiated
	Destination  uint8  // Destination domainID where message is to be sent
	DepositNonce uint64 // Nonce for the deposit
	ResourceId   types.ResourceID
	Metadata     message.Metadata
	Data         []byte
}
