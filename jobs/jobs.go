package jobs

import (
	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/comm/p2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
	"time"
)

func StartCommunicationHealthCheckJob(h host.Host) {
	healthComm := p2p.NewCommunication(h, "p2p/health")
	for {
		time.Sleep(5 * time.Minute)
		log.Info().Msg("Starting communication health check")
		communicationErrors := comm.ExecuteCommHealthCheck(healthComm, h.Peerstore().Peers())
		for _, cerr := range communicationErrors {
			log.Err(cerr).Msg("communication error")
		}
	}
}
