package depositHandlers

import (
	"errors"
	"math/big"

	"github.com/sygmaprotocol/sygma-core/relayer/message"

	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ChainSafe/sygma-relayer/chains/evm/executor"
)

type Erc721DepositHandler struct{}

// Erc721DepositHandler converts data pulled from ERC721 deposit event logs into message
func (dh *Erc721DepositHandler) HandleDeposit(sourceID, destId uint8, nonce uint64, resourceID [32]byte, calldata, handlerResponse []byte) (*message.Message, error) {
	if len(calldata) < 64 {
		err := errors.New("invalid calldata length: less than 84 bytes")
		return nil, err
	}

	// first 32 bytes are tokenId
	tokenId := calldata[:32]

	// 32 - 64 is recipient address length
	recipientAddressLength := big.NewInt(0).SetBytes(calldata[32:64])

	// 64 - (64 + recipient address length) is recipient address
	recipientAddress := calldata[64:(64 + recipientAddressLength.Int64())]

	// (64 + recipient address length) - ((64 + recipient address length) + 32) is metadata length
	metadataLength := big.NewInt(0).SetBytes(
		calldata[(64 + recipientAddressLength.Int64()):((64 + recipientAddressLength.Int64()) + 32)],
	)
	// ((64 + recipient address length) + 32) - ((64 + recipient address length) + 32 + metadata length) is metadata
	var metadata []byte
	var metadataStart int64
	if metadataLength.Cmp(big.NewInt(0)) == 1 {
		metadataStart = (64 + recipientAddressLength.Int64()) + 32
		metadata = calldata[metadataStart : metadataStart+metadataLength.Int64()]
	}
	// arbitrary metadata that will be most likely be used by the relayer
	var meta map[string]interface{}

	payload := []interface{}{
		tokenId,
		recipientAddress,
		metadata,
	}

	return chains.NewMessage(sourceID, destId, chains.TransferMessageData{
		DepositNonce: nonce,
		ResourceId:   resourceID,
		Metadata:     meta,
		Payload:      payload,
		Type:         executor.ERC721,
	}, "Transfer"), nil
}
