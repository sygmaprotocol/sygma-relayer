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
	metadata.Data["gasLimit"] = uint64(big.NewInt(0).SetBytes(maxFee).Int64())

	return message.NewMessage(sourceID, destId, nonce, resourceID, PermissionlessGenericTransfer, payload, metadata), nil
}

// Erc20DepositHandler converts data pulled from event logs into message.
// handlerResponse contains converted amount into 18 decimals if the token does not have 18 decimals.
func Erc20DepositHandler(sourceID, destId uint8, nonce uint64, resourceID types.ResourceID, calldata, handlerResponse []byte) (*message.Message, error) {
	if len(calldata) < 84 {
		err := errors.New("invalid calldata length: less than 84 bytes")
		return nil, err
	}

	// @dev
	// amount: first 32 bytes of calldata
	var amount []byte
	if len(handlerResponse) > 0 {
		amount = handlerResponse[:32]
	} else {
		amount = calldata[:32]
	}

	// lenRecipientAddress: second 32 bytes of calldata [32:64]
	// does not need to be derived because it is being calculated
	// within ERC20MessageHandler
	// https://github.com/ChainSafe/chainbridge-core/blob/main/chains/evm/voter/message-handler.go#L108

	// 32-64 is recipient address length
	recipientAddressLength := big.NewInt(0).SetBytes(calldata[32:64])

	// 64 - (64 + recipient address length) is recipient address
	recipientAddress := calldata[64:(64 + recipientAddressLength.Int64())]

	// if there is priority data, parse it and use it
	payload := []interface{}{
		amount,
		recipientAddress,
	}

	return message.NewMessage(sourceID, destId, nonce, resourceID, message.FungibleTransfer, payload, message.Metadata{}), nil
}
