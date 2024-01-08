package depositHandlers

import (
	"errors"
	"math/big"

	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type GenericDepositHandler struct{}

// GenericDepositHandler converts data pulled from generic deposit event logs into message
func (dh *GenericDepositHandler) HandleDeposit(sourceID, destId uint8, nonce uint64, resourceID [32]byte, calldata, handlerResponse []byte) (*message.Message, error) {
	if len(calldata) < 32 {
		err := errors.New("invalid calldata length: less than 32 bytes")
		return nil, err
	}

	// first 32 bytes are metadata length
	metadataLen := big.NewInt(0).SetBytes(calldata[:32])
	metadata := calldata[32 : 32+metadataLen.Int64()]
	payload := []interface{}{
		metadata,
	}

	// generic handler has specific payload length and doesn't support arbitrary metadata
	return chains.NewTransferMessage(sourceID, destId, nonce, resourceID, nil, payload, GenericTransfer), nil
}
