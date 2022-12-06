// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events_test

import (
	"errors"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	core_types "github.com/ChainSafe/chainbridge-core/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	"github.com/stretchr/testify/suite"
	"math/big"
	"testing"
)

var errIncorrectdeposit_dataLen = errors.New("invalid calldata length: less than 84 bytes")
var errNoCorrespondingDepositHandler = errors.New("no corresponding deposit handler for this transfer type exists")

type Erc20HandlerTestSuite struct {
	suite.Suite
}

func TestRunErc20HandlerTestSuite(t *testing.T) {
	suite.Run(t, new(Erc20HandlerTestSuite))
}

func (s *Erc20HandlerTestSuite) SetupSuite()    {}
func (s *Erc20HandlerTestSuite) TearDownSuite() {}
func (s *Erc20HandlerTestSuite) SetupTest()     {}
func (s *Erc20HandlerTestSuite) TearDownTest()  {}

func (s *Erc20HandlerTestSuite) TestErc20HandleEvent() {
	// Alice
	sender, _ := types.NewAccountIDFromHexString("0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d")

	// Bob
	recipientByteSlice := []byte{0x8e, 0xaf, 0x4, 0x15, 0x16, 0x87, 0x73, 0x63, 0x26, 0xc9, 0xfe, 0xa1, 0x7e, 0x25, 0xfc, 0x52, 0x87, 0x61, 0x36, 0x93, 0xc9, 0x12, 0x90, 0x9c, 0xb2, 0x26, 0xaa, 0x47, 0x94, 0xf2, 0x6a, 0x48}
	var calldata []byte
	amount, _ := types.BigIntToIntBytes(big.NewInt(2), 32)
	calldata = append(calldata, amount...)
	recipientLen, _ := (types.BigIntToIntBytes(big.NewInt(int64(len(recipientByteSlice))), 32))
	calldata = append(calldata, recipientLen...)
	calldata = append(calldata, types.Bytes(recipientByteSlice)...)

	depositLog := &events.Deposit{
		DestinationDomainID: 0,
		ResourceID:          [32]byte{0},
		DepositNonce:        1,
		SenderAddress:       *sender,
		Data:                calldata,
		HandlerResponse:     []byte{},
	}

	sourceID := uint8(1)
	amountParsed := calldata[:32]
	recipientAddressParsed := calldata[64:]

	expected := &message.Message{
		Source:       sourceID,
		Destination:  depositLog.DestinationDomainID,
		DepositNonce: depositLog.DepositNonce,
		ResourceId:   depositLog.ResourceID,
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			amountParsed,
			recipientAddressParsed,
		},
	}

	message, err := events.FungibleTransferHandler(sourceID, depositLog.DestinationDomainID, depositLog.DepositNonce, depositLog.ResourceID, depositLog.Data, depositLog.HandlerResponse)

	s.Nil(err)
	s.NotNil(message)
	s.Equal(message, expected)
}

func (s *Erc20HandlerTestSuite) TestErc20HandleEventWithPriority() {
	// Alice
	sender, _ := types.NewAccountIDFromHexString("0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d")

	//Bob
	recipientByteSlice := []byte{0x8e, 0xaf, 0x4, 0x15, 0x16, 0x87, 0x73, 0x63, 0x26, 0xc9, 0xfe, 0xa1, 0x7e, 0x25, 0xfc, 0x52, 0x87, 0x61, 0x36, 0x93, 0xc9, 0x12, 0x90, 0x9c, 0xb2, 0x26, 0xaa, 0x47, 0x94, 0xf2, 0x6a, 0x48}
	priority := uint8(1)
	var calldata []byte
	amount, _ := types.BigIntToIntBytes(big.NewInt(2), 32)
	calldata = append(calldata, amount...)
	recipientLen, _ := (types.BigIntToIntBytes(big.NewInt(int64(len(recipientByteSlice))), 32))
	calldata = append(calldata, recipientLen...)
	calldata = append(calldata, types.Bytes(recipientByteSlice)...)
	priorityLen, _ := (types.BigIntToIntBytes(big.NewInt(int64(len([]uint8{priority}))), 1))
	calldata = append(calldata, priorityLen...) // Length of priority
	calldata = append(calldata, priority)

	depositLog := &events.Deposit{
		DestinationDomainID: 0,
		ResourceID:          [32]byte{0},
		DepositNonce:        1,
		SenderAddress:       *sender,
		Data:                calldata,
		HandlerResponse:     []byte{},
	}

	sourceID := uint8(1)
	amountParsed := calldata[:32]
	// 32-64 is recipient address length
	recipientAddressLength := big.NewInt(0).SetBytes(calldata[32:64])

	// 64 - (64 + recipient address length) is recipient address
	recipientAddressParsed := calldata[64:(64 + recipientAddressLength.Int64())]
	expected := &message.Message{
		Source:       sourceID,
		Destination:  depositLog.DestinationDomainID,
		DepositNonce: depositLog.DepositNonce,
		ResourceId:   depositLog.ResourceID,
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			amountParsed,
			recipientAddressParsed,
		},
		Metadata: message.Metadata{},
	}

	message, err := events.FungibleTransferHandler(sourceID, depositLog.DestinationDomainID, depositLog.DepositNonce, depositLog.ResourceID, depositLog.Data, depositLog.HandlerResponse)

	s.Nil(err)
	s.NotNil(message)
	s.Equal(message, expected)
}

func (s *Erc20HandlerTestSuite) TestErc20HandleEventIncorrectdeposit_dataLen() {
	// Alice
	sender, _ := types.NewAccountIDFromHexString("0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d")

	metadeposit_data := []byte("0xdeadbeef")
	metadeposit_dataLen, _ := (types.BigIntToIntBytes(big.NewInt(int64(len(metadeposit_data))), 32))

	var calldata []byte
	calldata = append(calldata, metadeposit_dataLen...)
	calldata = append(calldata, metadeposit_data...)

	depositLog := &events.Deposit{
		DestinationDomainID: 0,
		ResourceID:          [32]byte{0},
		DepositNonce:        1,
		SenderAddress:       *sender,
		Data:                calldata,
		HandlerResponse:     []byte{},
	}

	sourceID := uint8(1)

	message, err := events.FungibleTransferHandler(sourceID, depositLog.DestinationDomainID, depositLog.DepositNonce, depositLog.ResourceID, depositLog.Data, depositLog.HandlerResponse)

	s.Nil(message)
	s.EqualError(err, errIncorrectdeposit_dataLen.Error())
}

func (s *Erc20HandlerTestSuite) TestSuccesfullyRegisterFungibleTransferHandler() {
	recipientByteSlice := []byte{0x8e, 0xaf, 0x4, 0x15, 0x16, 0x87, 0x73, 0x63, 0x26, 0xc9, 0xfe, 0xa1, 0x7e, 0x25, 0xfc, 0x52, 0x87, 0x61, 0x36, 0x93, 0xc9, 0x12, 0x90, 0x9c, 0xb2, 0x26, 0xaa, 0x47, 0x94, 0xf2, 0x6a, 0x48}
	// Create calldata
	var calldata []byte
	amount, _ := types.BigIntToIntBytes(big.NewInt(2), 32)
	calldata = append(calldata, amount...)
	recipientLen, _ := (types.BigIntToIntBytes(big.NewInt(int64(len(recipientByteSlice))), 32))
	calldata = append(calldata, recipientLen...)
	calldata = append(calldata, types.Bytes(recipientByteSlice)...)

	d1 := &events.Deposit{
		DepositNonce:        1,
		DestinationDomainID: 2,
		ResourceID:          core_types.ResourceID{},
		DepositType:         message.FungibleTransfer,
		HandlerResponse:     []byte{},
		Data:                calldata,
	}

	depositHandler := events.NewSubstrateDepositHandler()
	// Register FungibleTransferHandler function
	depositHandler.RegisterDepositHandler(message.FungibleTransfer, events.FungibleTransferHandler)
	message1, err1 := depositHandler.HandleDeposit(1, d1.DestinationDomainID, d1.DepositNonce, d1.ResourceID, d1.Data, d1.DepositType, d1.HandlerResponse)
	s.Nil(err1)
	s.NotNil(message1)

	// Use unregistered transfer type
	message2, err2 := depositHandler.HandleDeposit(1, d1.DestinationDomainID, d1.DepositNonce, d1.ResourceID, d1.Data, message.NonFungibleTransfer, d1.HandlerResponse)
	s.Nil(message2)
	s.NotNil(err2)
	s.EqualError(err2, errNoCorrespondingDepositHandler.Error())
}
