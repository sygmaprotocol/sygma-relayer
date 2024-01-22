// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor_test

import (
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/sygmaprotocol/sygma-core/relayer/message"

	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/events"
	"github.com/ChainSafe/sygma-relayer/chains/evm/executor"
	"github.com/ChainSafe/sygma-relayer/e2e/evm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

var errIncorrectERC20PayloadLen = errors.New("malformed payload. Len  of payload should be 2")
var errIncorrectERC721PayloadLen = errors.New("malformed payload. Len  of payload should be 3")
var errIncorrectGenericPayloadLen = errors.New("malformed payload. Len  of payload should be 1")

var errIncorrectAmount = errors.New("wrong payload amount format")
var errIncorrectRecipient = errors.New("wrong payload recipient format")
var errIncorrectTokenID = errors.New("wrong payload tokenID format")
var errIncorrectMetadata = errors.New("wrong payload metadata format")

// ERC20
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

func (s *Erc20HandlerTestSuite) TestErc20HandleMessage() {

	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: executor.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				[]byte{2}, // amount
				[]byte{241, 229, 143, 177, 119, 4, 194, 218, 132, 121, 165, 51, 249, 250, 212, 173, 9, 147, 202, 107}, // recipientAddress
			},
		},
		Type: executor.ERC20,
	}

	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(err)
	s.NotNil(prop)
}

func (s *Erc20HandlerTestSuite) TestErc20HandleMessageIncorrectDataLen() {
	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: executor.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				[]byte{2}, // amount
			},
		},
		Type: executor.ERC20,
	}

	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectERC20PayloadLen.Error())
}

func (s *Erc20HandlerTestSuite) TestErc20HandleMessageIncorrectAmount() {

	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: executor.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				"incorrectAmount", // amount
				[]byte{241, 229, 143, 177, 119, 4, 194, 218, 132, 121, 165, 51, 249, 250, 212, 173, 9, 147, 202, 107}, // recipientAddress
			},
		},
		Type: executor.ERC20,
	}

	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectAmount.Error())
}

func (s *Erc20HandlerTestSuite) TestErc20HandleMessageIncorrectRecipient() {
	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: executor.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				[]byte{2},            // amount
				"incorrectRecipient", // recipientAddress
			},
		},
		Type: executor.ERC20,
	}

	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectRecipient.Error())
}

// ERC721
type Erc721HandlerTestSuite struct {
	suite.Suite
}

func TestRunErc721HandlerTestSuite(t *testing.T) {
	suite.Run(t, new(Erc721HandlerTestSuite))
}

func (s *Erc721HandlerTestSuite) SetupSuite()    {}
func (s *Erc721HandlerTestSuite) TearDownSuite() {}
func (s *Erc721HandlerTestSuite) SetupTest()     {}
func (s *Erc721HandlerTestSuite) TearDownTest()  {}

func (s *Erc721HandlerTestSuite) TestErc721MessageHandlerEmptyMetadata() {

	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: executor.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				[]byte{2}, // tokenID
				[]byte{241, 229, 143, 177, 119, 4, 194, 218, 132, 121, 165, 51, 249, 250, 212, 173, 9, 147, 202, 107}, // recipientAddress
				[]byte{}, // metadata
			},
		},
		Type: executor.ERC721,
	}

	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(err)
	s.NotNil(prop)
}

func (s *Erc721HandlerTestSuite) TestErc721MessageHandlerIncorrectDataLen() {
	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: executor.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				[]byte{2}, // tokenID
			},
		},
		Type: executor.ERC721,
	}

	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectERC721PayloadLen.Error())
}

func (s *Erc721HandlerTestSuite) TestErc721MessageHandlerIncorrectAmount() {

	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: executor.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				"incorrectAmount", // tokenID
				[]byte{241, 229, 143, 177, 119, 4, 194, 218, 132, 121, 165, 51, 249, 250, 212, 173, 9, 147, 202, 107}, // recipientAddress
				[]byte{}, // metadata
			},
		},
		Type: executor.ERC721,
	}

	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectTokenID.Error())
}

func (s *Erc721HandlerTestSuite) TestErc721MessageHandlerIncorrectRecipient() {

	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: executor.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				[]byte{2}, // amount
				"incorrectRecipient",
				[]byte{}, // metadata
			},
		},
		Type: executor.ERC721,
	}

	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectRecipient.Error())
}

func (s *Erc721HandlerTestSuite) TestErc721MessageHandlerIncorrectMetadata() {

	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: executor.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				[]byte{2}, // amount
				[]byte{241, 229, 143, 177, 119, 4, 194, 218, 132, 121, 165, 51, 249, 250, 212, 173, 9, 147, 202, 107}, // recipientAddress
				"incorrectMetadata", // metadata
			},
		},
		Type: executor.ERC721,
	}

	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectMetadata.Error())
}

// GENERIC
type GenericHandlerTestSuite struct {
	suite.Suite
}

func TestRunGenericHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GenericHandlerTestSuite))
}

func (s *GenericHandlerTestSuite) SetupSuite()    {}
func (s *GenericHandlerTestSuite) TearDownSuite() {}
func (s *GenericHandlerTestSuite) SetupTest()     {}
func (s *GenericHandlerTestSuite) TearDownTest()  {}
func (s *GenericHandlerTestSuite) TestGenericHandleEvent() {
	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: executor.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				[]byte{}, // metadata
			},
		},
		Type: executor.PermissionedGeneric,
	}

	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(err)
	s.NotNil(prop)
}

func (s *GenericHandlerTestSuite) TestGenericHandleEventIncorrectDataLen() {
	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: executor.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload:      []interface{}{},
		},
		Type: executor.PermissionedGeneric,
	}

	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectGenericPayloadLen.Error())
}

func (s *GenericHandlerTestSuite) TestGenericHandleEventIncorrectMetadata() {
	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: executor.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				"incorrectMetadata", // metadata
			},
		},
		Type: executor.PermissionedGeneric,
	}
	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
	s.EqualError(err, errIncorrectMetadata.Error())
}

// Permissionless
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
	calldata := evm.ConstructPermissionlessGenericDepositData(
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
		Source:      sourceID,
		Destination: depositLog.DestinationDomainID,
		Data: executor.TransferMessageData{
			DepositNonce: depositLog.DepositNonce,
			ResourceId:   depositLog.ResourceID,
			Payload: []interface{}{
				functionSig,
				contractAddress.Bytes(),
				common.LeftPadBytes(maxFee.Bytes(), 32),
				depositor.Bytes(),
				hash,
			},
		},
		Type: executor.PermissionlessGeneric,
	}

	mh := executor.TransferMessageHandler{}
	prop, err := mh.HandleMessage(message)

	expectedData, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000030d4000001402091eeff969b33a5ce8a729dae325879bf76f90145c1f5961696bad2e73f73417f07ef55c62a2dc5b307868617368")
	expected := chains.NewProposal(
		message.Source,
		message.Destination,
		chains.TransferProposalData{
			DepositNonce: message.Data.(executor.TransferMessageData).DepositNonce,
			ResourceId:   message.Data.(executor.TransferMessageData).ResourceId,
			Metadata:     message.Data.(executor.TransferMessageData).Metadata,
			Data:         expectedData,
		},
		chains.TransferProposalType,
	)
	s.Nil(err)
	s.Equal(expected, prop)
}
