// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package depositHandlers_test

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/sygma-relayer/e2e/evm"
	"github.com/ChainSafe/sygma-relayer/relayer/transfer"
	"github.com/sygmaprotocol/sygma-core/relayer/message"

	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/events"
	"github.com/ChainSafe/sygma-relayer/chains/evm/listener/depositHandlers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/stretchr/testify/suite"
)

type GenericHandlerTestSuite struct {
	suite.Suite
}

func TestRunGenericHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GenericHandlerTestSuite))
}

func (s *GenericHandlerTestSuite) TestGenericHandleEventIncorrectDataLen() {
	metadata := []byte("0xdeadbeef")

	var calldata []byte
	calldata = append(calldata, math.PaddedBigBytes(big.NewInt(int64(len(metadata))), 16)...)
	calldata = append(calldata, metadata...)

	depositLog := &events.Deposit{
		DestinationDomainID: 0,
		ResourceID:          [32]byte{0},
		DepositNonce:        1,
		SenderAddress:       common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485"),
		Data:                calldata,
		HandlerResponse:     []byte{},
	}

	sourceID := uint8(1)

	genericDepositHandler := depositHandlers.GenericDepositHandler{}
	message, err := genericDepositHandler.HandleDeposit(
		sourceID,
		depositLog.DestinationDomainID,
		depositLog.DepositNonce,
		depositLog.ResourceID,
		depositLog.Data,
		depositLog.HandlerResponse,
		"messageID")

	s.Nil(message)
	s.EqualError(err, "invalid calldata length: less than 32 bytes")
}

func (s *GenericHandlerTestSuite) TestGenericHandleEventEmptyMetadata() {
	metadata := []byte("")
	calldata := evm.ConstructGenericDepositData(metadata)

	depositLog := &events.Deposit{
		DestinationDomainID: 0,
		ResourceID:          [32]byte{0},
		DepositNonce:        1,
		SenderAddress:       common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485"),
		Data:                calldata,
		HandlerResponse:     []byte{},
	}

	sourceID := uint8(1)
	expected := &message.Message{
		Source:      sourceID,
		Destination: depositLog.DestinationDomainID,
		Data: transfer.TransferMessageData{
			DepositNonce: depositLog.DepositNonce,
			ResourceId:   depositLog.ResourceID,
			Payload: []interface{}{
				metadata,
			},
			Type: transfer.PermissionedGenericTransfer,
		},

		Type: transfer.TransferMessageType,
		ID:   "messageID",
	}

	genericDepositHandler := depositHandlers.GenericDepositHandler{}
	message, err := genericDepositHandler.HandleDeposit(
		sourceID,
		depositLog.DestinationDomainID,
		depositLog.DepositNonce,
		depositLog.ResourceID,
		depositLog.Data,
		depositLog.HandlerResponse,
		"messageID")

	s.Nil(err)
	s.NotNil(message)
	s.Equal(message, expected)
}

func (s *GenericHandlerTestSuite) TestGenericHandleEvent() {
	metadata := []byte("0xdeadbeef")
	calldata := evm.ConstructGenericDepositData(metadata)

	depositLog := &events.Deposit{
		DestinationDomainID: 0,
		ResourceID:          [32]byte{0},
		DepositNonce:        1,
		SenderAddress:       common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485"),
		Data:                calldata,
		HandlerResponse:     []byte{},
	}

	sourceID := uint8(1)
	expected := &message.Message{
		Source:      sourceID,
		Destination: depositLog.DestinationDomainID,
		Data: transfer.TransferMessageData{
			DepositNonce: depositLog.DepositNonce,
			ResourceId:   depositLog.ResourceID,
			Payload: []interface{}{
				metadata,
			},
			Type: transfer.PermissionedGenericTransfer,
		},

		Type: transfer.TransferMessageType,
		ID:   "messageID",
	}

	genericDepositHandler := depositHandlers.GenericDepositHandler{}
	message, err := genericDepositHandler.HandleDeposit(
		sourceID,
		depositLog.DestinationDomainID,
		depositLog.DepositNonce,
		depositLog.ResourceID,
		depositLog.Data,
		depositLog.HandlerResponse,
		"messageID")

	s.Nil(err)
	s.NotNil(message)
	s.Equal(message, expected)
}
