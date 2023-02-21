// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package bridge

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/client"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/executor/proposal"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
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

type BridgePallet struct {
	client *client.SubstrateClient
}

func NewBridgePallet(
	client *client.SubstrateClient,
) *BridgePallet {
	return &BridgePallet{
		client: client,
	}
}

func (p *BridgePallet) ExecuteProposals(
	proposals []*proposal.Proposal,
	signature []byte,
) (*types.Hash, error) {
	bridgeProposals := make([]BridgeProposal, 0)
	for _, prop := range proposals {
		bridgeProposals = append(bridgeProposals, BridgeProposal{
			OriginDomainID: prop.Source,
			DepositNonce:   prop.DepositNonce,
			ResourceID:     prop.ResourceId,
			Data:           prop.Data,
		})
	}

	return p.client.Transact(
		"SygmaBridge.execute_proposal",
		bridgeProposals,
		signature,
	)
}

func (p *BridgePallet) ProposalsHash(proposals []*proposal.Proposal) ([]byte, error) {
	formattedProps := make([]interface{}, len(proposals))
	for i, prop := range proposals {
		formattedProps[i] = map[string]interface{}{
			"originDomainID": math.NewHexOrDecimal256(int64(prop.Source)),
			"depositNonce":   math.NewHexOrDecimal256(int64(prop.DepositNonce)),
			"resourceID":     hexutil.Encode(prop.ResourceId[:]),
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
			ChainId:           math.NewHexOrDecimal256(p.client.ChainID.Int64()),
			Version:           bridgeVersion,
			VerifyingContract: verifyingContract,
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

func (p *BridgePallet) IsProposalExecuted(prop *proposal.Proposal) (bool, error) {
	log.Debug().
		Str("depositNonce", strconv.FormatUint(prop.DepositNonce, 10)).
		Str("resourceID", hexutil.Encode(prop.ResourceId[:])).
		Msg("Getting is proposal executed")
	var res bool
	err := p.client.Conn.Call(res, "sygma_isProposalExecuted", big.NewInt(int64(prop.DepositNonce)), big.NewInt(int64(prop.Source)))
	if err != nil {
		return false, err
	}
	return res, nil
}
