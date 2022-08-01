package common_test

import (
	"testing"

	"github.com/ChainSafe/sygma/tss/common"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/suite"
)

type TssMessageTestSuite struct {
	suite.Suite
}

func TestRunTssMessageTestSuite(t *testing.T) {
	suite.Run(t, new(TssMessageTestSuite))
}

func (s *TssMessageTestSuite) Test_UnmarshaledMessageShouldBeEqual() {
	originalMsg := &common.TssMessage{
		MsgBytes:    []byte{1},
		IsBroadcast: true,
		From:        "fromAddress",
	}
	msgBytes, err := common.MarshalTssMessage(originalMsg.MsgBytes, originalMsg.IsBroadcast, originalMsg.From)
	s.Nil(err)

	unmarshaledMsg, err := common.UnmarshalTssMessage(msgBytes)
	s.Nil(err)

	s.Equal(originalMsg, unmarshaledMsg)
}

type StartMessageTestSuite struct {
	suite.Suite
}

func TestRunStartMessageTestSuite(t *testing.T) {
	suite.Run(t, new(StartMessageTestSuite))
}

func (s *StartMessageTestSuite) Test_UnmarshaledMessageShouldBeEqual() {
	originalMsg := &common.StartMessage{
		Params: []byte("test"),
	}
	msgBytes, err := common.MarshalStartMessage(originalMsg.Params)
	s.Nil(err)

	unmarshaledMsg, err := common.UnmarshalStartMessage(msgBytes)
	s.Nil(err)

	s.Equal(originalMsg, unmarshaledMsg)
}

type FailMessageTestSuite struct {
	suite.Suite
}

func TestRunFailMessageTestSuite(t *testing.T) {
	suite.Run(t, new(FailMessageTestSuite))
}

func (s *FailMessageTestSuite) Test_UnmarshaledMessageShouldBeEqual() {
	peerID1, _ := peer.Decode("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")
	peerID2, _ := peer.Decode("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")
	excludedPeers := []peer.ID{peerID1, peerID2}
	originalMsg := &common.FailMessage{
		ExcludedPeers: excludedPeers,
	}

	msgBytes, err := common.MarshalFailMessage(excludedPeers)
	s.Nil(err)

	unmarshaledMsg, err := common.UnmarshalFailMessage(msgBytes)
	s.Nil(err)

	s.Equal(originalMsg, unmarshaledMsg)
}
