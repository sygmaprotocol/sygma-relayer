package depositHandlers

import (
	"errors"
	"math/big"

	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ChainSafe/sygma-relayer/relayer/transfer"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type Erc20DepositHandler struct{}

// Erc20DepositHandler converts data pulled from event logs into message
// handlerResponse can be an empty slice
func (dh *Erc20DepositHandler) HandleDeposit(sourceID, destId uint8, nonce uint64, resourceID [32]byte, calldata, handlerResponse []byte) (*message.Message, error) {
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
	return chains.NewMessage(sourceID, destId, transfer.TransferMessageData{
		DepositNonce: nonce,
		ResourceId:   resourceID,
		Metadata:     nil,
		Payload:      payload,
		Type:         transfer.ERC20Message,
	}, transfer.TransferMessageType), nil
}
