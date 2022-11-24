// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/rs/zerolog/log"
)

type ChainConnection interface {
	UpdateMetatdata() error
}

type SystemUpdateEventHandler struct {
	conn ChainConnection
}

func NewSystemUpdateEventHandler(conn ChainConnection) *SystemUpdateEventHandler {
	return &SystemUpdateEventHandler{
		conn: conn,
	}
}

func (eh *SystemUpdateEventHandler) HandleEvents(evts Events, msgChan chan []*message.Message) error {
	if len(evts.System_CodeUpdated) > 0 {
		err := eh.conn.UpdateMetatdata()
		if err != nil {
			log.Error().Err(err).Msg("Unable to update Metadata")
			return err
		}
	}
	return nil
}
