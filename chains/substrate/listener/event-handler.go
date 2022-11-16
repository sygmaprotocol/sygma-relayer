package listener

import (
	"github.com/ChainSafe/chainbridge-core/relayer/message"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	"github.com/rs/zerolog/log"
)

// handleEvents calls the associated handler for all registered event types
func (l *SubstrateListener) handleEvents(domainID uint8, evts *events.Events) ([]*message.Message, error) {
	msgs := make([]*message.Message, 0)
	if l.eventHandlers[message.FungibleTransfer] != nil {
		for _, evt := range evts.ChainBridge_FungibleTransfer {
			m, err := l.eventHandlers[message.FungibleTransfer](domainID, evt)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, m)
		}
	}
	if l.eventHandlers[message.NonFungibleTransfer] != nil {
		for _, evt := range evts.ChainBridge_NonFungibleTransfer {
			m, err := l.eventHandlers[message.NonFungibleTransfer](domainID, evt)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, m)

		}
	}
	if l.eventHandlers[message.GenericTransfer] != nil {
		for _, evt := range evts.ChainBridge_GenericTransfer {
			m, err := l.eventHandlers[message.GenericTransfer](domainID, evt)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, m)
		}
	}
	if len(evts.System_CodeUpdated) > 0 {
		err := l.client.UpdateMetatdata()
		if err != nil {
			log.Error().Err(err).Msg("Unable to update Metadata")
			return nil, err
		}
	}
	return msgs, nil
}
