package proposal

import (
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/types"
	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func NewEvmProposal(source, destination uint8, depositNonce uint64, resourceId types.ResourceID, data []byte, handlerAddress, bridgeAddress common.Address, metadata message.Metadata) *EvmProposal {
	return &EvmProposal{
		Proposal: chains.Proposal{
			Source:       source,
			Destination:  destination,
			DepositNonce: depositNonce,
			ResourceId:   resourceId,
			Data:         data,
		},
		HandlerAddress: handlerAddress,
		BridgeAddress:  bridgeAddress,
		Metadata:       metadata,
	}
}

type EvmProposal struct {
	chains.Proposal
	Metadata       message.Metadata
	HandlerAddress common.Address
	BridgeAddress  common.Address
}

// GetDataHash constructs and returns Evmproposal data hash
func (p *EvmProposal) GetDataHash() common.Hash {
	return crypto.Keccak256Hash(append(p.HandlerAddress.Bytes(), p.Data...))
}

// GetID constructs Evmproposal unique identifier
func (p *EvmProposal) GetID() common.Hash {
	return crypto.Keccak256Hash(append([]byte{p.Source}, byte(p.DepositNonce)))
}
