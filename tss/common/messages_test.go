package common_test

import (
	"testing"

	"github.com/ChainSafe/chainbridge-core/tss/common"
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
		MsgBytes:   []byte{1},
		IsBrodcast: true,
		From:       "fromAddress",
	}
	msgBytes, err := common.MarshalTssMessage(originalMsg.MsgBytes, originalMsg.IsBrodcast, originalMsg.From)
	s.Nil(err)

	unmarshaledMsg, err := common.UnmarshalTssMessage(msgBytes)
	s.Nil(err)

	s.Equal(originalMsg, unmarshaledMsg)
}
