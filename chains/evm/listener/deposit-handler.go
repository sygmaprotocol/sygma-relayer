package listener

import (
	"errors"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/types"
)

const (
	PermissionlessGenericTransfer message.TransferType = "PermissionlessGenericTransfer"
)

// GenericDepositHandler converts data pulled from generic deposit event logs into message
func PermissionlessGenericDepositHandler(sourceID, destId uint8, nonce uint64, resourceID types.ResourceID, calldata, handlerResponse []byte) (*message.Message, error) {
	if len(calldata) < 32 {
		err := errors.New("invalid calldata length: less than 32 bytes")
		return nil, err
	}

	metadataLen := big.NewInt(0).SetBytes(calldata[:32])
	executeFunctionSignature := calldata[32:64]
	executeContractAddress := calldata[64:96]
	maxFee := calldata[96:128]
	metadataDepositor := calldata[128:160]
	executionData := calldata[160 : 128+metadataLen.Int64()]

	payload := []interface{}{
		executeFunctionSignature,
		executeContractAddress,
		maxFee,
		metadataDepositor,
		executionData,
	}

	return message.NewMessage(sourceID, destId, nonce, resourceID, PermissionlessGenericTransfer, payload, message.Metadata{}), nil
}
