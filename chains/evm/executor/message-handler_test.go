// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor_test

import (
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/chains/evm/executor/proposal"
	"github.com/ChainSafe/chainbridge-core/relayer/message"

	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/sygma-relayer/chains/evm/executor"
	"github.com/ChainSafe/sygma-relayer/chains/evm/listener"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

var errIncorrectPayloadLen = errors.New("malformed payload. Len  of payload should be 4")
var errIncorrectTokenIDs = errors.New("wrong payload tokenIDs format")
var errIncorrectAmounts = errors.New("wrong payload amounts format")
var errIncorrectRecipient = errors.New("wrong payload recipient format")
var errIncorrectRecipientLen = errors.New("malformed payload. Len  of recipient should be 20")
var errIncorrectTransferData = errors.New("wrong payload transferData format")

type PermissionlessGenericHandlerTestSuite struct {
	suite.Suite
}

func TestRunPermissionlessGenericHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(PermissionlessGenericHandlerTestSuite))
}

func (s *PermissionlessGenericHandlerTestSuite) Test_HandleMessage() {
	hash := []byte("0xhash")
	functionSig, _ := hex.DecodeString("0x654cf88c")
	contractAddress := common.HexToAddress("0x02091EefF969b33A5CE8A729DaE325879bf76f90")
	depositor := common.HexToAddress("0x5C1F5961696BaD2e73f73417f07EF55C62a2dC5b")
	maxFee := big.NewInt(200000)
	var metadata []byte
	metadata = append(metadata, hash[:]...)
	calldata := bridge.ConstructPermissionlessGenericDepositData(
		metadata,
		functionSig,
		contractAddress.Bytes(),
		depositor.Bytes(),
		maxFee,
	)
	depositLog := &events.Deposit{
		DestinationDomainID: 0,
		ResourceID:          [32]byte{0},
		DepositNonce:        1,
		SenderAddress:       common.HexToAddress("0x5C1F5961696BaD2e73f73417f07EF55C62a2dC5b"),
		Data:                calldata,
		HandlerResponse:     []byte{},
	}
	sourceID := uint8(1)
	message := &message.Message{
		Source:       sourceID,
		Destination:  depositLog.DestinationDomainID,
		DepositNonce: depositLog.DepositNonce,
		ResourceId:   depositLog.ResourceID,
		Type:         listener.PermissionlessGenericTransfer,
		Payload: []interface{}{
			functionSig,
			contractAddress.Bytes(),
			common.LeftPadBytes(maxFee.Bytes(), 32),
			depositor.Bytes(),
			hash,
		},
	}
	handlerAddr := common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485")
	bridgeAddr := common.HexToAddress("0xf1e58fb17704c2da8479a533f9fad4ad0993ca6b")

	prop, err := executor.PermissionlessGenericMessageHandler(
		message,
		handlerAddr,
		bridgeAddr,
	)

	expectedData, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000030d4000001402091eeff969b33a5ce8a729dae325879bf76f90145c1f5961696bad2e73f73417f07ef55c62a2dc5b307868617368")
	expected := proposal.NewProposal(
		message.Source,
		message.Destination,
		message.DepositNonce,
		message.ResourceId,
		expectedData,
		handlerAddr,
		bridgeAddr,
		message.Metadata,
	)
	s.Nil(err)
	s.Equal(expected, prop)
}

// Erc1155
type Erc1155HandlerTestSuite struct {
	suite.Suite
}

func TestRunErc1155HandlerTestSuite(t *testing.T) {
	suite.Run(t, new(Erc1155HandlerTestSuite))
}

func (s *Erc1155HandlerTestSuite) SetupSuite()    {}
func (s *Erc1155HandlerTestSuite) TearDownSuite() {}
func (s *Erc1155HandlerTestSuite) SetupTest()     {}
func (s *Erc1155HandlerTestSuite) TearDownTest()  {}

func (s *Erc1155HandlerTestSuite) Test_HandleErc1155Message() {
	message := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]*big.Int{
				big.NewInt(2),
			},
			[]*big.Int{
				big.NewInt(3),
			},
			[]byte{28, 58, 3, 208, 76, 2, 107, 31, 75, 66, 8, 210, 206, 5, 60, 86, 134, 230, 251, 141},
			[]byte{},
		},
	}

	prop, err := executor.Erc1155MessageHandler(message, common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485"), common.HexToAddress("0xf1e58fb17704c2da8479a533f9fad4ad0993ca6b"))

	s.Nil(err)
	s.NotNil(prop)
}

func (s *Erc1155HandlerTestSuite) Test_HandleErc1155Message_InvalidPayloadLen() {
	message := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]*big.Int{
				big.NewInt(2),
			},
			[]*big.Int{
				big.NewInt(3),
			},
			[]byte{28, 58, 3, 208, 76, 2, 107, 31, 75, 66, 8, 210, 206, 5, 60, 86, 134, 230, 251, 141},
		},
	}

	prop, err := executor.Erc1155MessageHandler(message, common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485"), common.HexToAddress("0xf1e58fb17704c2da8479a533f9fad4ad0993ca6b"))

	s.Nil(prop)
	s.EqualError(err, errIncorrectPayloadLen.Error())
}

func (s *Erc1155HandlerTestSuite) Test_HandleErc1155Message_InvalidTokenIDs() {
	message := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			2,
			[]*big.Int{
				big.NewInt(3),
			},
			[]byte{28, 58, 3, 208, 76, 2, 107, 31, 75, 66, 8, 210, 206, 5, 60, 86, 134, 230, 251, 141},
			[]byte{},
		},
	}

	prop, err := executor.Erc1155MessageHandler(message, common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485"), common.HexToAddress("0xf1e58fb17704c2da8479a533f9fad4ad0993ca6b"))

	s.Nil(prop)
	s.EqualError(err, errIncorrectTokenIDs.Error())
}

func (s *Erc1155HandlerTestSuite) Test_HandleErc1155Message_InvalidAmounts() {
	message := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]*big.Int{
				big.NewInt(2),
			},
			3,
			[]byte{28, 58, 3, 208, 76, 2, 107, 31, 75, 66, 8, 210, 206, 5, 60, 86, 134, 230, 251, 141},
			[]byte{},
		},
	}

	prop, err := executor.Erc1155MessageHandler(message, common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485"), common.HexToAddress("0xf1e58fb17704c2da8479a533f9fad4ad0993ca6b"))

	s.Nil(prop)
	s.EqualError(err, errIncorrectAmounts.Error())
}

func (s *Erc1155HandlerTestSuite) Test_HandleErc1155Message_InvalidRecipient() {
	message := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]*big.Int{
				big.NewInt(2),
			},
			[]*big.Int{
				big.NewInt(3),
			},
			"invalidRecipient",
			[]byte{},
		},
	}

	prop, err := executor.Erc1155MessageHandler(message, common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485"), common.HexToAddress("0xf1e58fb17704c2da8479a533f9fad4ad0993ca6b"))

	s.Nil(prop)
	s.EqualError(err, errIncorrectRecipient.Error())
}

func (s *Erc1155HandlerTestSuite) Test_HandleErc1155Message_InvalidRecipientLen() {
	message := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]*big.Int{
				big.NewInt(2),
			},
			[]*big.Int{
				big.NewInt(3),
			},
			[]byte{28, 58, 3, 208, 76, 2, 107, 31, 75, 66, 8, 210, 206, 5, 60, 86, 134, 230, 251},
			[]byte{},
		},
	}

	prop, err := executor.Erc1155MessageHandler(message, common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485"), common.HexToAddress("0xf1e58fb17704c2da8479a533f9fad4ad0993ca6b"))

	s.Nil(prop)
	s.EqualError(err, errIncorrectRecipientLen.Error())
}

func (s *Erc1155HandlerTestSuite) Test_HandleErc1155Message_InvalidTransferData() {
	message := &message.Message{
		Source:       1,
		Destination:  0,
		DepositNonce: 1,
		ResourceId:   [32]byte{0},
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			[]*big.Int{
				big.NewInt(2),
			},
			[]*big.Int{
				big.NewInt(3),
			},
			[]byte{28, 58, 3, 208, 76, 2, 107, 31, 75, 66, 8, 210, 206, 5, 60, 86, 134, 230, 251, 141},
			"invalidTransferData",
		},
	}

	prop, err := executor.Erc1155MessageHandler(message, common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485"), common.HexToAddress("0xf1e58fb17704c2da8479a533f9fad4ad0993ca6b"))

	s.Nil(prop)
	s.EqualError(err, errIncorrectTransferData.Error())
}
