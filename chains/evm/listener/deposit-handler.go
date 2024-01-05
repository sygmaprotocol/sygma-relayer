// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

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

func TransferDepositHandler(sourceID, destId uint8, nonce uint64, resourceID types.ResourceID, calldata, handlerResponse []byte) (*message.Message, error) {
	return nil, nil
}

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

func GenericDepositHandler(sourceID, destId uint8, nonce uint64, resourceID types.ResourceID, calldata, handlerResponse []byte) (*message.Message, error) {
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
	meta := message.Metadata{}
	return message.NewMessage(sourceID, destId, nonce, resourceID, message.GenericTransfer, payload, meta), nil
}

// Erc721DepositHandler converts data pulled from ERC721 deposit event logs into message
func Erc721DepositHandler(sourceID, destId uint8, nonce uint64, resourceID types.ResourceID, calldata, handlerResponse []byte) (*message.Message, error) {
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
	var meta message.Metadata

	payload := []interface{}{
		tokenId,
		recipientAddress,
		metadata,
	}

	if 64+recipientAddressLength.Int64()+32+metadataLength.Int64() < int64(len(calldata)) {
		// (metadataStart + metadataLength) - (metadataStart + metadataLength + 1) is priority length
		priorityLength := big.NewInt(0).SetBytes(calldata[(64 + recipientAddressLength.Int64() + 32 + metadataLength.Int64()):(64 + recipientAddressLength.Int64() + 32 + metadataLength.Int64() + 1)])
		// (metadataStart + metadataLength + 1) - (metadataStart + metadataLength + 1) + priority length) is priority data
		priority := calldata[(64 + recipientAddressLength.Int64() + 32 + metadataLength.Int64() + 1):(64 + recipientAddressLength.Int64() + 32 + metadataLength.Int64() + 1 + priorityLength.Int64())]
		meta.Priority = priority[0]
	}
	return message.NewMessage(sourceID, destId, nonce, resourceID, message.NonFungibleTransfer, payload, meta), nil
}
