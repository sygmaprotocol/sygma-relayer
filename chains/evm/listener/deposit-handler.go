package listener

import (
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/types"
)

const (
	PermissionlessGenericTransfer message.TransferType = "PermissionlessGenericTransfer"
)

// GenericDepositHandler converts data pulled from generic deposit event logs into message
func PermissionlessGenericDepositHandler(sourceID, destId uint8, nonce uint64, resourceID types.ResourceID, calldata, handlerResponse []byte) (*message.Message, error) {
	maxFee := calldata[:32]

	functionSigLen := big.NewInt(0).SetBytes(calldata[32:34])
	functionSigEnd := 34 + functionSigLen.Int64()
	functionSig := calldata[34:functionSigEnd]

	contractAddressLen := big.NewInt(0).SetBytes(calldata[functionSigEnd : functionSigEnd+1])
	contractAddressEnd := functionSigEnd + 1 + contractAddressLen.Int64()
	contractAddress := calldata[functionSigEnd+1 : contractAddressEnd]

	depositorLen := big.NewInt(0).SetBytes(calldata[contractAddressEnd : contractAddressEnd+1])
	depositorEnd := contractAddressEnd + 1 + depositorLen.Int64()
	depositor := calldata[contractAddressEnd+1 : depositorEnd]

	executionData := calldata[depositorEnd:]

	payload := []interface{}{
		functionSig,
		contractAddress,
		maxFee,
		depositor,
		executionData,
	}

	metadata := message.Metadata{}
	metadata.Data["fee"] = uint64(big.NewInt(0).SetBytes(maxFee).Int64())

	return message.NewMessage(sourceID, destId, nonce, resourceID, PermissionlessGenericTransfer, payload, metadata), nil
}
