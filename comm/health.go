package comm

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
	"time"
)

const HealthTimeout = 10 * time.Second

func ExecuteCommHealthCheck(communication Communication, peers peer.IDSlice) []*CommunicationError {
	errChan := make(chan error)
	endTimer := time.NewTimer(HealthTimeout)

	communication.Broadcast(peers, []byte{}, Unknown, "health-session", errChan)

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
