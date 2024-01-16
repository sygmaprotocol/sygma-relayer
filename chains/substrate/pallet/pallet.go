// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package pallet

import (
	"strconv"

	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/author"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

const bridgeVersion = "3.1.0"
const verifyingContract = "6CdE2Cd82a4F8B74693Ff5e194c19CA08c2d1c68"

type BridgeProposal struct {
	OriginDomainID uint8
	DepositNonce   uint64
	ResourceID     [32]byte
	Data           []byte
}

type Pallet struct {
	*client.SubstrateClient
}

func NewPallet(
	client *client.SubstrateClient,
) *Pallet {
	return &Pallet{
		client,
	}
}

func (p *Pallet) ExecuteProposals(
	proposals []*proposal.Proposal,
	signature []byte,
) (types.Hash, *author.ExtrinsicStatusSubscription, error) {
	bridgeProposals := make([]BridgeProposal, 0)
	for _, prop := range proposals {
		bridgeProposals = append(bridgeProposals, BridgeProposal{
			OriginDomainID: prop.Source,
			DepositNonce:   prop.Data.(chains.TransferProposalData).DepositNonce,
			ResourceID:     prop.Data.(chains.TransferProposalData).ResourceId,
			Data:           prop.Data.(chains.TransferProposalData).Data,
		})
	}

	return p.Transact(
		"SygmaBridge.execute_proposal",
		bridgeProposals,
		signature,
	)
}

func (p *Pallet) ProposalsHash(proposals []*proposal.Proposal) ([]byte, error) {
	return chains.ProposalsHash(proposals, p.ChainID.Int64(), verifyingContract, bridgeVersion)
}

func (p *Pallet) IsProposalExecuted(prop *proposal.Proposal) (bool, error) {

	transferProposal := &chains.TransferProposal{
		Source:      prop.Source,
		Destination: prop.Destination,
		Data: chains.TransferProposalData{
			DepositNonce: prop.Data.(chains.TransferProposalData).DepositNonce,
			ResourceId:   prop.Data.(chains.TransferProposalData).ResourceId,
			Metadata:     prop.Data.(chains.TransferProposalData).Metadata,
			Data:         prop.Data.(chains.TransferProposalData).Data,
		},
		Type: prop.Type,
	}

	log.Debug().
		Str("depositNonce", strconv.FormatUint(transferProposal.Data.DepositNonce, 10)).
		Str("resourceID", hexutil.Encode(transferProposal.Data.ResourceId[:])).
		Msg("Getting is proposal executed")
	var res bool
	err := p.Conn.Call(&res, "sygma_isProposalExecuted", transferProposal.Data.DepositNonce, transferProposal.Source)
	if err != nil {
		return false, err
	}

	return res, nil
}
