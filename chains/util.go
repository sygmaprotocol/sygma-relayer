package chains

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// CalculateStartingBlock returns first block number (smaller or equal) that is dividable with block confirmations
func CalculateStartingBlock(startBlock *big.Int, blockConfirmations *big.Int) *big.Int {
	mod := big.NewInt(0).Mod(startBlock, blockConfirmations)
	startBlock.Sub(startBlock, mod)
	return startBlock
}

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

func NewProposalsHash(proposals []*Proposal, chainID int64, verifContract string, bridgeVersion string) ([]byte, error) {
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
