package proposal

import (
	"github.com/ChainSafe/chainbridge-core/types"
)

func NewProposal(source, destination uint8, depositNonce uint64, resourceId types.ResourceID, data []byte) *Proposal {
	return &Proposal{
		Source:       source,
		DepositNonce: depositNonce,
		ResourceId:   resourceId,
		Destination:  destination,
		Data:         data,
	}
}

type Proposal struct {
	Source       uint8            // Source domainID where message was initiated
	DepositNonce uint64           // Nonce for the deposit
	ResourceId   types.ResourceID // change id -> ID
	Data         []byte
	Destination  uint8 // Destination domainID where message is to be sent
}
