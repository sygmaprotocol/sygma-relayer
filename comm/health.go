// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package comm

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
)

const HealthTimeout = 10 * time.Second

func ExecuteCommHealthCheck(communication Communication, peers peer.IDSlice) []*CommunicationError {
	errChan := make(chan error)
	endTimer := time.NewTimer(HealthTimeout)
	sessionID := "health-session"

	defer communication.CloseSession(sessionID)
	go communication.Broadcast(peers, []byte{}, Unknown, sessionID, errChan)

	var collectedErrors []*CommunicationError
	for {
		select {
		case err := <-errChan:
			switch err := err.(type) {
			case *CommunicationError:
				collectedErrors = append(collectedErrors, err)
			default:
				log.Err(err).Msg("Unknown error on checking communication health")
			}
		case <-endTimer.C:
			if len(collectedErrors) == 0 {
				log.Info().Msg("Communication healthy - successfully dialed all peers")
			} else {
				for _, e := range collectedErrors {
					log.Info().Err(e.Err).Msgf("Unable to broadcast to peer %s", e.Peer)
				}
			}
			return collectedErrors
		}
	}
}
