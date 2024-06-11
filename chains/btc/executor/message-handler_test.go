package executor_test

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/sygma-relayer/chains/btc/executor"
	"github.com/ChainSafe/sygma-relayer/relayer/transfer"
	"github.com/stretchr/testify/suite"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type BtcMessageHandlerTestSuite struct {
	suite.Suite
}

func TestRunBtcMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(BtcMessageHandlerTestSuite))
}

func (s *BtcMessageHandlerTestSuite) Test_ERC20HandleMessage_ValidMessage() {
	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: transfer.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				big.NewInt(100000045678).Bytes(), // amount
				[]byte("tb1pffdrehs8455lgnwquggf4dzf6jduz8v7d2usflyujq4ggh4jaapqpfjj83"),
			},
			Type: transfer.FungibleTransfer,
		},
		Type: transfer.TransferMessageType,
	}

	mh := executor.BtcMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(err)
	s.NotNil(prop)
	s.Equal(prop, &proposal.Proposal{
		Source:      1,
		Destination: 0,
		Data: executor.BtcTransferProposalData{
			Amount:       10,
			Recipient:    "tb1pffdrehs8455lgnwquggf4dzf6jduz8v7d2usflyujq4ggh4jaapqpfjj83",
			DepositNonce: 1,
		},
		Type: transfer.TransferProposalType,
	})
}

func (s *BtcMessageHandlerTestSuite) Test_ERC20HandleMessage_ValidMessage_Dust() {
	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: transfer.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				big.NewInt(1).Bytes(), // amount
				[]byte("tb1pffdrehs8455lgnwquggf4dzf6jduz8v7d2usflyujq4ggh4jaapqpfjj83"),
			},
			Type: transfer.FungibleTransfer,
		},
		Type: transfer.TransferMessageType,
	}

	mh := executor.BtcMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(err)
	s.NotNil(prop)
	s.Equal(prop, &proposal.Proposal{
		Source:      1,
		Destination: 0,
		Data: executor.BtcTransferProposalData{
			Amount:       0,
			Recipient:    "tb1pffdrehs8455lgnwquggf4dzf6jduz8v7d2usflyujq4ggh4jaapqpfjj83",
			DepositNonce: 1,
		},
		Type: transfer.TransferProposalType,
	})
}

func (s *BtcMessageHandlerTestSuite) Test_ERC20HandleMessage_IncorrectDataLen() {
	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: transfer.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				[]byte{2}, // amount
			},
			Type: transfer.FungibleTransfer,
		},
		Type: transfer.TransferMessageType,
	}

	mh := executor.BtcMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
}

func (s *BtcMessageHandlerTestSuite) Test_ERC20HandleMessage_IncorrectAmount() {

	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: transfer.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				"incorrectAmount", // amount
				[]byte{241, 229, 143, 177, 119, 4, 194, 218, 132, 121, 165, 51, 249, 250, 212, 173, 9, 147, 202, 107}, // recipientAddress
			},
			Type: transfer.FungibleTransfer,
		},
		Type: transfer.TransferMessageType,
	}

	mh := executor.BtcMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
}

func (s *BtcMessageHandlerTestSuite) Test_ERC20HandleMessage_IncorrectRecipient() {
	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: transfer.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				[]byte{2},            // amount
				"incorrectRecipient", // recipientAddress
			},
			Type: transfer.FungibleTransfer,
		},
		Type: transfer.TransferMessageType,
	}

	mh := executor.BtcMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
}

func (s *BtcMessageHandlerTestSuite) Test_HandleMessage_InvalidType() {
	message := &message.Message{
		Source:      1,
		Destination: 0,
		Data: transfer.TransferMessageData{
			DepositNonce: 1,
			ResourceId:   [32]byte{0},
			Payload: []interface{}{
				[]byte{2},            // amount
				"incorrectRecipient", // recipientAddress
			},
			Type: transfer.NonFungibleTransfer,
		},
		Type: transfer.TransferMessageType,
	}

	mh := executor.BtcMessageHandler{}
	prop, err := mh.HandleMessage(message)

	s.Nil(prop)
	s.NotNil(err)
}
