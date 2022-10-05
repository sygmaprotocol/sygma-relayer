package executor

import (
	"bytes"
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
	metadataDepositor, ok := msg.Payload[3].([]byte)
	if !ok {
		return nil, errors.New("wrong metadata depositor format")
	}
	executionData, ok := msg.Payload[4].([]byte)
	if !ok {
		return nil, errors.New("wrong execution data format")
	}

	data := bytes.Buffer{}
	metadataLen := big.NewInt(int64(len(executionData) + 32)).Bytes()
	data.Write(common.LeftPadBytes(metadataLen, 32))              // length of metadata (uint256)
	data.Write(common.LeftPadBytes(executeFunctionSignature, 32)) // bytes4
	data.Write(common.LeftPadBytes(executeContractAddress, 32))   // bytes32
	data.Write(common.LeftPadBytes(maxFee, 32))                   // uint256
	data.Write(common.LeftPadBytes(metadataDepositor, 32))        // bytes32
	data.Write(executionData)                                     // bytes

	return proposal.NewProposal(msg.Source, msg.Destination, msg.DepositNonce, msg.ResourceId, data.Bytes(), handlerAddr, bridgeAddress, msg.Metadata), nil
}
