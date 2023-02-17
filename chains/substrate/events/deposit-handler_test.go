// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events_test

import (
	"errors"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	core_types "github.com/ChainSafe/chainbridge-core/types"
	"github.com/centrifuge/go-substrate-rpc-client/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"

	"math/big"
	"testing"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	"github.com/stretchr/testify/suite"
)

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
	var substratePK = signature.KeyringPair{
		URI:       "//Alice",
		PublicKey: []byte{0xd4, 0x35, 0x93, 0xc7, 0x15, 0xfd, 0xd3, 0x1c, 0x61, 0x14, 0x1a, 0xbd, 0x4, 0xa9, 0x9f, 0xd6, 0x82, 0x2c, 0x85, 0x58, 0x85, 0x4c, 0xcd, 0xe3, 0x9a, 0x56, 0x84, 0xe7, 0xa5, 0x6d, 0xa2, 0x7d},
		Address:   "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
	}
	// Bob
	recipientByteSlice := []byte{0x8e, 0xaf, 0x4, 0x15, 0x16, 0x87, 0x73, 0x63, 0x26, 0xc9, 0xfe, 0xa1, 0x7e, 0x25, 0xfc, 0x52, 0x87, 0x61, 0x36, 0x93, 0xc9, 0x12, 0x90, 0x9c, 0xb2, 0x26, 0xaa, 0x47, 0x94, 0xf2, 0x6a, 0x48}
	var calldata []byte
	amount, _ := types.BigIntToIntBytes(big.NewInt(2), 32)
	calldata = append(calldata, amount...)
	recipientLen, _ := (types.BigIntToIntBytes(big.NewInt(int64(len(recipientByteSlice))), 32))
	calldata = append(calldata, recipientLen...)
	calldata = append(calldata, types.Bytes(recipientByteSlice)...)

	sender, _ := types.NewAccountID(substratePK.PublicKey)
	depositLog := &events.Deposit{
		Phase:        types.Phase{},
		DestDomainID: types.NewU8(2),
		ResourceID:   types.Bytes32{1},
		DepositNonce: types.NewU64(1),
		Sender:       *sender,
		TransferType: [1]byte{0},
		CallData:     calldata,
		Handler:      [1]byte{0},
		Topics:       []types.Hash{},
	}

	sourceID := uint8(1)
	amountParsed := calldata[:32]
	recipientAddressParsed := calldata[64:]

	expected := &message.Message{
		Source:       sourceID,
		Destination:  uint8(depositLog.DestDomainID),
		DepositNonce: uint64(depositLog.DepositNonce),
		ResourceId:   core_types.ResourceID(depositLog.ResourceID),
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			amountParsed,
			recipientAddressParsed,
		},
	}

	message, err := events.FungibleTransferHandler(sourceID, depositLog.DestDomainID, depositLog.DepositNonce, depositLog.ResourceID, depositLog.CallData)

	s.Nil(err)
	s.NotNil(message)
	s.Equal(message, expected)
}

func (s *Erc20HandlerTestSuite) TestErc20HandleEventIncorrectdeposit_dataLen() {
	// Alice
	var substratePK = signature.KeyringPair{
		URI:       "//Alice",
		PublicKey: []byte{0xd4, 0x35, 0x93, 0xc7, 0x15, 0xfd, 0xd3, 0x1c, 0x61, 0x14, 0x1a, 0xbd, 0x4, 0xa9, 0x9f, 0xd6, 0x82, 0x2c, 0x85, 0x58, 0x85, 0x4c, 0xcd, 0xe3, 0x9a, 0x56, 0x84, 0xe7, 0xa5, 0x6d, 0xa2, 0x7d},
		Address:   "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
	}
	metadeposit_data := []byte("0xdeadbeef")
	metadeposit_dataLen, _ := (types.BigIntToIntBytes(big.NewInt(int64(len(metadeposit_data))), 32))

	var calldata []byte
	calldata = append(calldata, metadeposit_dataLen...)
	calldata = append(calldata, metadeposit_data...)

	sender, _ := types.NewAccountID(substratePK.PublicKey)
	depositLog := &events.Deposit{
		Phase:        types.Phase{},
		DestDomainID: types.NewU8(2),
		ResourceID:   types.Bytes32{1},
		DepositNonce: types.NewU64(1),
		Sender:       *sender,
		TransferType: [1]byte{0},
		CallData:     calldata,
		Handler:      [1]byte{0},
		Topics:       []types.Hash{},
	}

	sourceID := uint8(1)
	amountParsed := calldata[:32]
	recipientAddressParsed := calldata[64:]

	expected := &message.Message{
		Source:       sourceID,
		Destination:  uint8(depositLog.DestDomainID),
		DepositNonce: uint64(depositLog.DepositNonce),
		ResourceId:   core_types.ResourceID(depositLog.ResourceID),
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			amountParsed,
			recipientAddressParsed,
		},
	}

	message, err := events.FungibleTransferHandler(sourceID, depositLog.DestDomainID, depositLog.DepositNonce, depositLog.ResourceID, depositLog.CallData)

	s.Nil(err)
	s.NotNil(message)
	s.Equal(message, expected)
}

func (s *Erc20HandlerTestSuite) TestSuccesfullyRegisterFungibleTransferHandler() {
	// Alice
	var substratePK = signature.KeyringPair{
		URI:       "//Alice",
		PublicKey: []byte{0xd4, 0x35, 0x93, 0xc7, 0x15, 0xfd, 0xd3, 0x1c, 0x61, 0x14, 0x1a, 0xbd, 0x4, 0xa9, 0x9f, 0xd6, 0x82, 0x2c, 0x85, 0x58, 0x85, 0x4c, 0xcd, 0xe3, 0x9a, 0x56, 0x84, 0xe7, 0xa5, 0x6d, 0xa2, 0x7d},
		Address:   "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
	}
	recipientByteSlice := []byte{0x8e, 0xaf, 0x4, 0x15, 0x16, 0x87, 0x73, 0x63, 0x26, 0xc9, 0xfe, 0xa1, 0x7e, 0x25, 0xfc, 0x52, 0x87, 0x61, 0x36, 0x93, 0xc9, 0x12, 0x90, 0x9c, 0xb2, 0x26, 0xaa, 0x47, 0x94, 0xf2, 0x6a, 0x48}
	// Create calldata
	var calldata []byte
	amount, _ := types.BigIntToIntBytes(big.NewInt(2), 32)
	calldata = append(calldata, amount...)
	recipientLen, _ := (types.BigIntToIntBytes(big.NewInt(int64(len(recipientByteSlice))), 32))
	calldata = append(calldata, recipientLen...)
	calldata = append(calldata, types.Bytes(recipientByteSlice)...)
	sender, _ := types.NewAccountID(substratePK.PublicKey)

	d1 := &events.Deposit{
		Phase:        types.Phase{},
		DestDomainID: types.NewU8(2),
		ResourceID:   types.Bytes32{1},
		DepositNonce: types.NewU64(1),
		Sender:       *sender,
		TransferType: [1]byte{0},
		CallData:     calldata,
		Handler:      [1]byte{0},
		Topics:       []types.Hash{},
	}

	depositHandler := events.NewSubstrateDepositHandler()
	// Register FungibleTransferHandler function
	depositHandler.RegisterDepositHandler(message.FungibleTransfer, events.FungibleTransferHandler)
	message1, err1 := depositHandler.HandleDeposit(1, d1.DestDomainID, d1.DepositNonce, d1.ResourceID, d1.CallData, d1.TransferType)
	s.Nil(err1)
	s.NotNil(message1)

	// Use unregistered transfer type
	message2, err2 := depositHandler.HandleDeposit(1, d1.DestDomainID, d1.DepositNonce, d1.ResourceID, d1.CallData, [1]byte{1})
	s.Nil(message2)
	s.NotNil(err2)
	s.EqualError(err2, errNoCorrespondingDepositHandler.Error())
}
