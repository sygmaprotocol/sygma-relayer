package eventHandlers

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ChainSafe/sygma-relayer/relayer/retry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type RetryEventHandler struct {
	log           zerolog.Logger
	eventListener EventListener
	retryAddress  common.Address
	domainID      uint8
	msgChan       chan []*message.Message
}

func NewRetryEventHandler(
	logC zerolog.Context,
	eventListener EventListener,
	retryAddress common.Address,
	domainID uint8,
	msgChan chan []*message.Message,
) *RetryEventHandler {
	return &RetryEventHandler{
		log:           logC.Logger(),
		eventListener: eventListener,
		retryAddress:  retryAddress,
		domainID:      domainID,
		msgChan:       msgChan,
	}
}

func (eh *RetryEventHandler) HandleEvents(
	startBlock *big.Int,
	endBlock *big.Int,
) error {
	retryEvents, err := eh.eventListener.FetchRetryEvents(context.Background(), eh.retryAddress, startBlock, endBlock)
	if err != nil {
		return fmt.Errorf("unable to fetch retry v2 events because of: %+v", err)
	}

	for _, e := range retryEvents {
		messageID := fmt.Sprintf("retry-%d-%d", e.SourceDomainID, e.DestinationDomainID)
		msg := message.NewMessage(
			eh.domainID,
			e.SourceDomainID,
			retry.RetryMessageData{
				SourceDomainID:      e.SourceDomainID,
				DestinationDomainID: e.DestinationDomainID,
				BlockHeight:         e.BlockHeight,
				ResourceID:          e.ResourceID,
			},
			messageID,
			retry.RetryMessageType,
		)

		eh.log.Info().Str("messageID", messageID).Msgf(
			"Resolved retry message %+v in block range: %s-%s", msg, startBlock.String(), endBlock.String(),
		)
		go func() { eh.msgChan <- []*message.Message{msg} }()
	}
	return nil
}
