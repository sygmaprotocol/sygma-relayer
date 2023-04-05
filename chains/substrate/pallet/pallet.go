// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package pallet

import (
	"math/big"
	"strconv"

	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/author"

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
	proposals []*chains.Proposal,
	signature []byte,
) (*author.ExtrinsicStatusSubscription, error) {
	bridgeProposals := make([]BridgeProposal, 0)
	for _, prop := range proposals {
		bridgeProposals = append(bridgeProposals, BridgeProposal{
			OriginDomainID: prop.OriginDomainID,
			DepositNonce:   prop.DepositNonce,
			ResourceID:     prop.ResourceID,
			Data:           prop.Data,
		})
	}

	return p.Transact(
		"SygmaBridge.execute_proposal",
		bridgeProposals,
		signature,
	)
}

func (p *Pallet) ProposalsHash(proposals []*chains.Proposal) ([]byte, error) {
	return chains.ProposalsHash(proposals, p.ChainID.Int64(), verifyingContract, bridgeVersion)
}

func (p *Pallet) IsProposalExecuted(prop *chains.Proposal) (bool, error) {
	log.Debug().
		Str("depositNonce", strconv.FormatUint(prop.DepositNonce, 10)).
		Str("resourceID", hexutil.Encode(prop.ResourceID[:])).
		Msg("Getting is proposal executed")
	var res bool
	err := p.Conn.Call(res, "sygma_isProposalExecuted", big.NewInt(int64(prop.DepositNonce)), big.NewInt(int64(prop.OriginDomainID)))
	if err != nil {
		return false, err
	}
	return res, nil
}
