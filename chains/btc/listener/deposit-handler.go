// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ChainSafe/sygma-relayer/relayer/transfer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type BtcDepositHandler struct{}

// NewBtcDepositHandler creates an instance of BtcDepositHandler that contains
// handler functions for processing deposit events
func NewBtcDepositHandler() *BtcDepositHandler {
	return &BtcDepositHandler{}
}

func (e *BtcDepositHandler) HandleDeposit(sourceID uint8,
	depositNonce uint64,
	resourceID [32]byte,
	amount *big.Int,
	data string,
	blockNumber *big.Int,
) (*message.Message, error) {

	// data is composed of recieverEVMAddress_destinationDomainID
	parsedData := strings.Split(data, "_")
	evmAdd := common.HexToAddress(parsedData[0]).Bytes()
	destDomainID, err := strconv.ParseUint(parsedData[1], 10, 8)
	if err != nil {
		return nil, err
	}

	payload := []interface{}{
		amount.Bytes(),
		evmAdd,
	}

	messageID := fmt.Sprintf("%d-%d-%d", sourceID, destDomainID, blockNumber)
	return message.NewMessage(sourceID, uint8(destDomainID), transfer.TransferMessageData{
		DepositNonce: depositNonce,
		ResourceId:   RESOURCE_ID,
		Metadata:     nil,
		Payload:      payload,
		Type:         transfer.FungibleTransfer,
	}, messageID, transfer.TransferMessageType), nil
}
