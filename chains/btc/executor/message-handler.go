package executor

import (
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type BtcProposalData struct {
	Amount    int64
	Recipient string
}

type BtcMessageHandler struct{}

func NewBtcMessageHandler() *BtcMessageHandler {
	return &BtcMessageHandler{}
}

func (mh *BtcMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	return &proposal.Proposal{}, nil
}
