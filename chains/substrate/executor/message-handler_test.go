package executor_test

import (
	"errors"
	"testing"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/executor"
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
	message := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]byte{2}, // amount
			[]byte{0x8e, 0xaf, 0x4, 0x15, 0x16, 0x87, 0x73, 0x63, 0x26, 0xc9, 0xfe, 0xa1, 0x7e, 0x25, 0xfc, 0x52, 0x87, 0x61, 0x36, 0x93, 0xc9, 0x12, 0x90, 0x9c, 0xb2, 0x26, 0xaa, 0x47, 0x94, 0xf2, 0x6a, 0x48}, // recipientAddress
		},
		Metadata: message.Metadata{
			Priority: uint8(1),
		},
	}

	prop, err := executor.FungibleTransferMessageHandler(message)

	s.Nil(err)
	s.NotNil(prop)
}

func (s *FungibleTransferHandlerTestSuite) TestFungibleTransferHandleMessageIncorrectDataLen() {
	message := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]byte{2}, // amount
		},
		Metadata: message.Metadata{
			Priority: uint8(1),
		},
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
		Metadata: message.Metadata{
			Priority: uint8(1),
		},
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
		Metadata: message.Metadata{
			Priority: uint8(1),
		},
	}

	prop, err := executor.FungibleTransferMessageHandler(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectRecipient.Error())
}

func (s *FungibleTransferHandlerTestSuite) TestSuccesfullyRegisterFungibleTransferMessageHandler() {
	messageData := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]byte{2}, // amount
			[]byte{0x8e, 0xaf, 0x4, 0x15, 0x16, 0x87, 0x73, 0x63, 0x26, 0xc9, 0xfe, 0xa1, 0x7e, 0x25, 0xfc, 0x52, 0x87, 0x61, 0x36, 0x93, 0xc9, 0x12, 0x90, 0x9c, 0xb2, 0x26, 0xaa, 0x47, 0x94, 0xf2, 0x6a, 0x48}, // recipientAddress
		},
		Metadata: message.Metadata{
			Priority: uint8(1),
		},
	}

	invalidMessageData := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.NonFungibleTransfer,
		Payload: []interface{}{
			[]byte{2}, // amount
			[]byte{0x8e, 0xaf, 0x4, 0x15, 0x16, 0x87, 0x73, 0x63, 0x26, 0xc9, 0xfe, 0xa1, 0x7e, 0x25, 0xfc, 0x52, 0x87, 0x61, 0x36, 0x93, 0xc9, 0x12, 0x90, 0x9c, 0xb2, 0x26, 0xaa, 0x47, 0x94, 0xf2, 0x6a, 0x48}, // recipientAddress
		},
		Metadata: message.Metadata{
			Priority: uint8(1),
		},
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
