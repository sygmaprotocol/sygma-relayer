// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package listener_test

import (
	"errors"
	"unsafe"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	core_types "github.com/ChainSafe/chainbridge-core/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"

	"math/big"
	"testing"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/listener"
	"github.com/ChainSafe/sygma-relayer/e2e/substrate"
	"github.com/stretchr/testify/suite"
)

var errNoCorrespondingDepositHandler = errors.New("no corresponding deposit handler for this transfer type exists")
var errIncorrectDataLen = errors.New("invalid calldata length: less than 84 bytes")

type Erc20HandlerTestSuite struct {
	suite.Suite
}

func TestRunErc20HandlerTestSuite(t *testing.T) {
	suite.Run(t, new(Erc20HandlerTestSuite))
}

func (s *Erc20HandlerTestSuite) TestErc20HandleEvent() {
	recipientAddr := *(*[]types.U8)(unsafe.Pointer(&substrate.SubstratePK.PublicKey))
	recipient := substrate.ConstructRecipientData(recipientAddr)

	var calldata []byte
	amount, _ := types.BigIntToIntBytes(big.NewInt(2), 32)
	calldata = append(calldata, amount...)
	recipientLen, _ := (types.BigIntToIntBytes(big.NewInt(int64(len(recipient))), 32))
	calldata = append(calldata, recipientLen...)
	calldata = append(calldata, types.Bytes(recipient)...)

	sender, _ := types.NewAccountID(substrate.SubstratePK.PublicKey)
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

	message, err := listener.FungibleTransferHandler(sourceID, depositLog.DestDomainID, depositLog.DepositNonce, depositLog.ResourceID, depositLog.CallData)

	s.Nil(err)
	s.NotNil(message)
	s.Equal(message, expected)
}

func (s *Erc20HandlerTestSuite) TestErc20HandleEventIncorrectdeposit_dataLen() {
	var calldata []byte

	sender, _ := types.NewAccountID(substrate.SubstratePK.PublicKey)
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

	message, err := listener.FungibleTransferHandler(sourceID, depositLog.DestDomainID, depositLog.DepositNonce, depositLog.ResourceID, depositLog.CallData)
	s.Nil(message)
	s.EqualError(err, errIncorrectDataLen.Error())
}

func (s *Erc20HandlerTestSuite) TestSuccesfullyRegisterFungibleTransferHandler() {
	recipientAddr := *(*[]types.U8)(unsafe.Pointer(&substrate.SubstratePK.PublicKey))
	recipient := substrate.ConstructRecipientData(recipientAddr)
	// Create calldata
	var calldata []byte
	amount, _ := types.BigIntToIntBytes(big.NewInt(2), 32)
	calldata = append(calldata, amount...)
	recipientLen, _ := (types.BigIntToIntBytes(big.NewInt(int64(len(recipient))), 32))
	calldata = append(calldata, recipientLen...)
	calldata = append(calldata, types.Bytes(recipient)...)
	sender, _ := types.NewAccountID(substrate.SubstratePK.PublicKey)

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

	depositHandler := listener.NewSubstrateDepositHandler()
	// Register FungibleTransferHandler function
	depositHandler.RegisterDepositHandler(message.FungibleTransfer, listener.FungibleTransferHandler)
	message1, err1 := depositHandler.HandleDeposit(1, d1.DestDomainID, d1.DepositNonce, d1.ResourceID, d1.CallData, d1.TransferType)
	s.Nil(err1)
	s.NotNil(message1)

	// Use unregistered transfer type
	message2, err2 := depositHandler.HandleDeposit(1, d1.DestDomainID, d1.DepositNonce, d1.ResourceID, d1.CallData, [1]byte{1})
	s.Nil(message2)
	s.NotNil(err2)
	s.EqualError(err2, errNoCorrespondingDepositHandler.Error())
}
