package listener

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/types"
)

const (
	PermissionlessGenericTransfer message.TransferType = "PermissionlessGenericTransfer"
)

// GenericDepositHandler converts data pulled from generic deposit event logs into message
func PermissionlessGenericDepositHandler(sourceID, destId uint8, nonce uint64, resourceID types.ResourceID, calldata, handlerResponse []byte) (*message.Message, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered from ", r)
		}
	}()

	maxFee := calldata[:32]

	functionSigLen := big.NewInt(0).SetBytes(calldata[32:34])
	functionSigEnd := 34 + functionSigLen.Int64()
	functionSig := calldata[34:functionSigEnd]

	contractAddressLen := big.NewInt(0).SetBytes(calldata[functionSigEnd : functionSigEnd+1])
	contractAddressEnd := functionSigEnd + 1 + contractAddressLen.Int64()
	contractAddress := calldata[functionSigEnd+1 : contractAddressEnd]

	executionData := calldata[contractAddressEnd+1:]

	payload := []interface{}{
		functionSig,
		contractAddress,
		maxFee,
		executionData,
	}

	metadata := message.Metadata{
		Data: make(map[string]interface{}),
	}
	metadata.Data["fee"] = uint64(big.NewInt(0).SetBytes(maxFee).Int64())

	return message.NewMessage(sourceID, destId, nonce, resourceID, PermissionlessGenericTransfer, payload, metadata), nil
}
