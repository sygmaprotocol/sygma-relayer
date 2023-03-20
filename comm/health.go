// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package comm

import (
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
)

const HealthTimeout = 10 * time.Second

func ExecuteCommHealthCheck(host host.Host, communication Communication, peers peer.IDSlice) []*CommunicationError {
	errChan := make(chan error)
	endTimer := time.NewTimer(HealthTimeout)
	sessionID := "health-session"

	if !isInTopology(host.ID(), peers) {
		log.Info().Msg("Relayer not in peer subset. Waiting for refresh...")
	}

	defer communication.CloseSession(sessionID)
	communication.Broadcast(peers, []byte{}, Unknown, sessionID, errChan)

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

func isInTopology(peer peer.ID, peers peer.IDSlice) bool {
	for _, id := range peers {
		if id == peer {
			return true
		}
	}

	return false
}
