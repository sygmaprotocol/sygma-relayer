// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"errors"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	PermissionlessGenericTransfer message.TransferType = "PermissionlessGenericTransfer"
	ERC1155Transfer               message.TransferType = "Erc1155"
)

// GenericDepositHandler converts data pulled from generic deposit event logs into message
func PermissionlessGenericDepositHandler(sourceID, destId uint8, nonce uint64, resourceID types.ResourceID, calldata, handlerResponse []byte) (*message.Message, error) {
	if len(calldata) < 76 {
		err := errors.New("invalid calldata length: less than 76 bytes")
		return nil, err
	}

	maxFee := calldata[:32]

	functionSigLen := big.NewInt(0).SetBytes(calldata[32:34])
	functionSigEnd := 34 + functionSigLen.Int64()
	functionSig := calldata[34:functionSigEnd]

	contractAddressLen := big.NewInt(0).SetBytes(calldata[functionSigEnd : functionSigEnd+1])
	contractAddressEnd := functionSigEnd + 1 + contractAddressLen.Int64()
	contractAddress := calldata[functionSigEnd+1 : contractAddressEnd]

	depositorLen := big.NewInt(0).SetBytes(calldata[contractAddressEnd : contractAddressEnd+1])
	depositorEnd := contractAddressEnd + 1 + depositorLen.Int64()
	depositorAddress := calldata[contractAddressEnd+1 : depositorEnd]
	executionData := calldata[depositorEnd:]

	payload := []interface{}{
		functionSig,
		contractAddress,
		maxFee,
		depositorAddress,
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

	// amount: first 32 bytes of calldata
	// handlerResponse: 32 bytes amount if handler converted amounts
	var amount []byte
	if len(handlerResponse) > 0 {
		amount = handlerResponse[:32]
	} else {
		amount = calldata[:32]
	}

	recipientAddressLength := big.NewInt(0).SetBytes(calldata[32:64])
	recipientAddress := calldata[64:(64 + recipientAddressLength.Int64())]
	payload := []interface{}{
		amount,
		recipientAddress,
	}

	return message.NewMessage(sourceID, destId, nonce, resourceID, message.FungibleTransfer, payload, message.Metadata{}), nil
}

func GetErc1155Type() (abi.Arguments, error) {
	tokenIDsType, err := abi.NewType("uint256[]", "", nil)
	if err != nil {
		return nil, err
	}

	amountsType, err := abi.NewType("uint256[]", "", nil)
	if err != nil {
		return nil, err
	}

	recipientType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	transferDataType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	// Define the arguments using the created types
	return abi.Arguments{
		abi.Argument{Name: "tokenIDs", Type: tokenIDsType, Indexed: false},
		abi.Argument{Name: "amounts", Type: amountsType, Indexed: false},
		abi.Argument{Name: "recipient", Type: recipientType, Indexed: false},
		abi.Argument{Name: "transferData", Type: transferDataType, Indexed: false},
	}, nil
}

func Erc1155DepositHandler(sourceID, destId uint8, nonce uint64, resourceID types.ResourceID, calldata, handlerResponse []byte) (*message.Message, error) {

	erc1155Type, err := GetErc1155Type()
	if err != nil {
		return nil, err
	}

	decodedCallData, err := erc1155Type.UnpackValues(calldata)
	if err != nil {
		return nil, err
	}

	payload := []interface{}{
		decodedCallData[0],
		decodedCallData[1],
		decodedCallData[2],
		decodedCallData[3],
	}

	return message.NewMessage(sourceID, destId, nonce, resourceID, ERC1155Transfer, payload, message.Metadata{}), nil
}
