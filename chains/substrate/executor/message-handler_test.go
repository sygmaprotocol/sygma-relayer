// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor_test

import (
	"encoding/hex"
	"errors"
	"testing"
	"unsafe"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/executor"
	"github.com/ChainSafe/sygma-relayer/e2e/substrate"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"

	"github.com/stretchr/testify/suite"
)

var errIncorrectFungibleTransferPayloadLen = errors.New("malformed payload. Len  of payload should be 2")
var errIncorrectAmount = errors.New("wrong payload amount format")
var errIncorrectRecipient = errors.New("wrong payload recipient format")

type FungibleTransferHandlerTestSuite struct {
	suite.Suite
}

func TestRunFungibleTransferHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(FungibleTransferHandlerTestSuite))
}

func (s *FungibleTransferHandlerTestSuite) TestFungibleTransferHandleMessage() {
	recipientAddr := *(*[]types.U8)(unsafe.Pointer(&substrate.SubstratePK.PublicKey))
	recipient := substrate.ConstructRecipientData(recipientAddr)

	message := &message.Message{
		Source:       1,
		Destination:  2,
		DepositNonce: 1,
		ResourceId:   [32]byte{1},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]byte{2}, // amount
			recipient,
		},
		Metadata: message.Metadata{},
	}
	data, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000002400010100d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d")
	expectedProp := &chains.Proposal{
		OriginDomainID: 1,
		Destination:    2,
		DepositNonce:   1,
		ResourceID:     [32]byte{1},
		Data:           data,
	}

	prop, err := executor.FungibleTransferMessageHandler(message)

	s.Nil(err)
	s.Equal(prop, expectedProp)
}

func (s *FungibleTransferHandlerTestSuite) TestFungibleTransferHandleMessageIncorrectDataLen() {
	message := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{1},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]byte{2}, // amount
		},
		Metadata: message.Metadata{},
	}

	prop, err := executor.FungibleTransferMessageHandler(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectFungibleTransferPayloadLen.Error())
}

func (s *FungibleTransferHandlerTestSuite) TestFungibleTransferHandleMessageIncorrectAmount() {
	message := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			"incorrectAmount", // amount
			[]byte{0x8e, 0xaf, 0x4, 0x15, 0x16, 0x87, 0x73, 0x63, 0x26, 0xc9, 0xfe, 0xa1, 0x7e, 0x25, 0xfc, 0x52, 0x87, 0x61, 0x36, 0x93, 0xc9, 0x12, 0x90, 0x9c, 0xb2, 0x26, 0xaa, 0x47, 0x94, 0xf2, 0x6a, 0x48}, // recipientAddress
		},
		Metadata: message.Metadata{},
	}

	prop, err := executor.FungibleTransferMessageHandler(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectAmount.Error())
}

func (s *FungibleTransferHandlerTestSuite) TestFungibleTransferHandleMessageIncorrectRecipient() {
	message := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]byte{2},            // amount
			"incorrectRecipient", // recipientAddress
		},
		Metadata: message.Metadata{},
	}

	prop, err := executor.FungibleTransferMessageHandler(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectRecipient.Error())
}

func (s *FungibleTransferHandlerTestSuite) TestSuccesfullyRegisterFungibleTransferMessageHandler() {
	recipientAddr := *(*[]types.U8)(unsafe.Pointer(&substrate.SubstratePK.PublicKey))
	recipient := substrate.ConstructRecipientData(recipientAddr)

	messageData := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]byte{2}, // amount
			recipient,
		},
		Metadata: message.Metadata{},
	}

	invalidMessageData := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.NonFungibleTransfer,
		Payload: []interface{}{
			[]byte{2}, // amount
			recipient,
		},
		Metadata: message.Metadata{},
	}

	depositMessageHandler := executor.NewSubstrateMessageHandler()
	// Register FungibleTransferMessageHandler function
	depositMessageHandler.RegisterMessageHandler(message.FungibleTransfer, executor.FungibleTransferMessageHandler)
	prop1, err1 := depositMessageHandler.HandleMessage(messageData)
	s.Nil(err1)
	s.NotNil(prop1)

	// Use unregistered transfer type
	prop2, err2 := depositMessageHandler.HandleMessage(invalidMessageData)
	s.Nil(prop2)
	s.NotNil(err2)
}
