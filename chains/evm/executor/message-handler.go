// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"bytes"
	"encoding/gob"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ChainSafe/chainbridge-core/chains/evm/executor/proposal"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
)

func PermissionlessGenericMessageHandler(msg *message.Message, handlerAddr, bridgeAddress common.Address) (*proposal.Proposal, error) {
	executeFunctionSignature, ok := msg.Payload[0].([]byte)
	if !ok {
		return nil, errors.New("wrong function signature format")
	}
	executeContractAddress, ok := msg.Payload[1].([]byte)
	if !ok {
		return nil, errors.New("wrong contract address format")
	}
	maxFee, ok := msg.Payload[2].([]byte)
	if !ok {
		return nil, errors.New("wrong max fee format")
	}
	depositor, ok := msg.Payload[3].([]byte)
	if !ok {
		return nil, errors.New("wrong depositor data format")
	}
	executionData, ok := msg.Payload[4].([]byte)
	if !ok {
		return nil, errors.New("wrong execution data format")
	}

	data := bytes.Buffer{}
	data.Write(common.LeftPadBytes(maxFee, 32))

	data.Write(common.LeftPadBytes(big.NewInt(int64(len(executeFunctionSignature))).Bytes(), 2))
	data.Write(executeFunctionSignature)

	data.Write([]byte{byte(len(executeContractAddress))})
	data.Write(executeContractAddress)

	data.Write([]byte{byte(len(depositor))})
	data.Write(depositor)

	data.Write(executionData)

	return proposal.NewProposal(msg.Source, msg.Destination, msg.DepositNonce, msg.ResourceId, data.Bytes(), handlerAddr, bridgeAddress, msg.Metadata), nil
}

func Erc1155MessageHandler(msg *message.Message, handlerAddr, bridgeAddress common.Address) (*proposal.Proposal, error) {

	if len(msg.Payload) != 4 {
		return nil, errors.New("malformed payload. Len  of payload should be 4")
	}
	tokenIDs, ok := msg.Payload[0].([]*big.Int)
	if !ok {
		return nil, errors.New("wrong payload tokenIDs format")
	}
	amounts, ok := msg.Payload[1].([]*big.Int)
	if !ok {
		return nil, errors.New("wrong payload amounts format")
	}
	recipient, ok := msg.Payload[2].([]byte)
	if !ok {
		return nil, errors.New("wrong payload recipient format")
	}
	transferData, ok := msg.Payload[3].([]byte)
	if !ok {
		return nil, errors.New("wrong payload transferData format")
	}

	// Convert []*big.Int slices to []byte
	tokenIDsBytes, err := encodeBigIntSlice(tokenIDs)
	if err != nil {
		return nil, err
	}

	amountsBytes, err := encodeBigIntSlice(amounts)
	if err != nil {
		return nil, err
	}

	// Concatenate the byte slices
	data := bytes.Buffer{}
	data.Write(tokenIDsBytes)
	data.Write(amountsBytes)
	data.Write(recipient)
	data.Write(transferData)

	return proposal.NewProposal(msg.Source, msg.Destination, msg.DepositNonce, msg.ResourceId, data.Bytes(), handlerAddr, bridgeAddress, msg.Metadata), nil
}

func encodeBigIntSlice(ints []*big.Int) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(ints)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
