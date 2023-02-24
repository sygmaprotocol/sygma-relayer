package chains

import (
	"fmt"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func NewProposal(source, destination uint8, depositNonce uint64, resourceId types.ResourceID, data []byte, metadata message.Metadata) *Proposal {
	return &Proposal{
		OriginDomainID: source,
		DepositNonce:   depositNonce,
		ResourceID:     resourceId,
		Destination:    destination,
		Data:           data,
		Metadata:       metadata,
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

func ProposalsHash(proposals []*Proposal, chainID int64, verifContract string, bridgeVersion string) ([]byte, error) {
	formattedProps := make([]interface{}, len(proposals))
	for i, prop := range proposals {
		formattedProps[i] = map[string]interface{}{
			"originDomainID": math.NewHexOrDecimal256(int64(prop.OriginDomainID)),
			"depositNonce":   math.NewHexOrDecimal256(int64(prop.DepositNonce)),
			"resourceID":     hexutil.Encode(prop.ResourceID[:]),
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
