// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ChainSafe/chainbridge-core/chains/evm/executor/proposal"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/evm/listener"
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
	_, ok := msg.Payload[0].([]*big.Int)
	if !ok {
		return nil, errors.New("wrong payload tokenIDs format")
	}
	_, ok = msg.Payload[1].([]*big.Int)
	if !ok {
		return nil, errors.New("wrong payload amounts format")
	}
	_, ok = msg.Payload[2].([]byte)
	if !ok {
		return nil, errors.New("wrong payload recipient format")
	}
	if len(msg.Payload[2].([]byte)) != 20 {
		return nil, errors.New("malformed payload. Len  of recipient should be 20")
	}
	_, ok = msg.Payload[3].([]byte)
	if !ok {
		return nil, errors.New("wrong payload transferData format")
	}

	data, err := listener.Erc1155DepositData.Encode(msg.Payload)
	if err != nil {
		return nil, err
	}

	return proposal.NewProposal(msg.Source, msg.Destination, msg.DepositNonce, msg.ResourceId, data, handlerAddr, bridgeAddress, msg.Metadata), nil
}
