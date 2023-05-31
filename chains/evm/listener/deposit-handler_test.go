// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener_test

import (
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/deposit"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/relayer/message"

	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/sygma-relayer/chains/evm/listener"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/stretchr/testify/suite"
)

type PermissionlessGenericHandlerTestSuite struct {
	suite.Suite
}

func TestRunPermissionlessGenericHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(PermissionlessGenericHandlerTestSuite))
}

func (s *PermissionlessGenericHandlerTestSuite) TestHandleEvent() {
	hash := []byte("0xhash")
	functionSig, _ := hex.DecodeString("654cf88c")
	contractAddress := common.HexToAddress("0x02091EefF969b33A5CE8A729DaE325879bf76f90")
	depositor := common.HexToAddress("0x5C1F5961696BaD2e73f73417f07EF55C62a2dC5b")
	maxFee := big.NewInt(200000)
	var executionData []byte
	executionData = append(executionData, hash[:]...)
	metadata := make(map[string]interface{})
	metadata["gasLimit"] = uint64(200000)

	calldata := bridge.ConstructPermissionlessGenericDepositData(
		hash[:],
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
	expected := &message.Message{
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
			executionData,
		},
		Metadata: message.Metadata{
			Data: metadata,
		},
	}

	message, err := listener.PermissionlessGenericDepositHandler(
		sourceID,
		depositLog.DestinationDomainID,
		depositLog.DepositNonce,
		depositLog.ResourceID,
		depositLog.Data,
		depositLog.HandlerResponse,
	)

	s.Nil(err)
	s.NotNil(message)
	s.Equal(message, expected)
}

func (s *PermissionlessGenericHandlerTestSuite) Test_HandleEvent_IncorrectDataLen() {
	metadata := make(map[string]interface{})
	metadata["gasLimit"] = uint64(200000)
	var calldata []byte
	calldata = append(calldata, math.PaddedBigBytes(big.NewInt(int64(len(metadata))), 32)...)
	depositLog := &events.Deposit{
		DestinationDomainID: 0,
		ResourceID:          [32]byte{0},
		DepositNonce:        1,
		SenderAddress:       common.HexToAddress("0x5C1F5961696BaD2e73f73417f07EF55C62a2dC5b"),
		Data:                calldata,
		HandlerResponse:     []byte{},
	}

	sourceID := uint8(1)
	_, err := listener.PermissionlessGenericDepositHandler(
		sourceID,
		depositLog.DestinationDomainID,
		depositLog.DepositNonce,
		depositLog.ResourceID,
		depositLog.Data,
		depositLog.HandlerResponse,
	)

	s.NotNil(err)
}

var errIncorrectDataLen = errors.New("invalid calldata length: less than 84 bytes")

type Erc20HandlerTestSuite struct {
	suite.Suite
}

func TestRunErc20HandlerTestSuite(t *testing.T) {
	suite.Run(t, new(Erc20HandlerTestSuite))
}

func (s *Erc20HandlerTestSuite) TestErc20HandleEvent() {
	// 0xf1e58fb17704c2da8479a533f9fad4ad0993ca6b
	recipientByteSlice := []byte{241, 229, 143, 177, 119, 4, 194, 218, 132, 121, 165, 51, 249, 250, 212, 173, 9, 147, 202, 107}

	calldata := deposit.ConstructErc20DepositData(recipientByteSlice, big.NewInt(2))
	depositLog := &events.Deposit{
		DestinationDomainID: 0,
		ResourceID:          [32]byte{0},
		DepositNonce:        1,
		SenderAddress:       common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485"),
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

	message, err := listener.Erc20DepositHandler(sourceID, depositLog.DestinationDomainID, depositLog.DepositNonce, depositLog.ResourceID, depositLog.Data, depositLog.HandlerResponse)

	s.Nil(err)
	s.NotNil(message)
	s.Equal(message, expected)
}

func (s *Erc20HandlerTestSuite) TestErc20HandleEventIncorrectDataLen() {
	metadata := []byte("0xdeadbeef")

	var calldata []byte
	calldata = append(calldata, math.PaddedBigBytes(big.NewInt(int64(len(metadata))), 32)...)
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

	message, err := listener.Erc20DepositHandler(sourceID, depositLog.DestinationDomainID, depositLog.DepositNonce, depositLog.ResourceID, depositLog.Data, depositLog.HandlerResponse)

	s.Nil(message)
	s.EqualError(err, errIncorrectDataLen.Error())
}

func (s *Erc20HandlerTestSuite) TestErc20HandleEventAlternativeAmount() {
	// 0xf1e58fb17704c2da8479a533f9fad4ad0993ca6b
	recipientByteSlice := []byte{241, 229, 143, 177, 119, 4, 194, 218, 132, 121, 165, 51, 249, 250, 212, 173, 9, 147, 202, 107}

	var handlerResponse []byte
	handlerResponse = append(handlerResponse, math.PaddedBigBytes(big.NewInt(200), 32)...) // Amount (ERC20)
	calldata := deposit.ConstructErc20DepositData(recipientByteSlice, big.NewInt(2))
	depositLog := &events.Deposit{
		DestinationDomainID: 0,
		ResourceID:          [32]byte{0},
		DepositNonce:        1,
		SenderAddress:       common.HexToAddress("0x4CEEf6139f00F9F4535Ad19640Ff7A0137708485"),
		Data:                calldata,
		HandlerResponse:     handlerResponse,
	}

	sourceID := uint8(1)
	recipientAddressParsed := calldata[64:]

	expected := &message.Message{
		Source:       sourceID,
		Destination:  depositLog.DestinationDomainID,
		DepositNonce: depositLog.DepositNonce,
		ResourceId:   depositLog.ResourceID,
		Type:         message.FungibleTransfer,
		Payload: []interface{}{
			handlerResponse[:32],
			recipientAddressParsed,
		},
	}

	message, err := listener.Erc20DepositHandler(sourceID, depositLog.DestinationDomainID, depositLog.DepositNonce, depositLog.ResourceID, depositLog.Data, depositLog.HandlerResponse)

	s.Nil(err)
	s.NotNil(message)
	s.Equal(message, expected)
	s.Equal(big.NewInt(200), new(big.Int).SetBytes(message.Payload[0].([]byte)))
}
