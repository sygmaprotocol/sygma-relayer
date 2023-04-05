package jobs

import (
	"time"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/comm/p2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
)

type Metrics interface {
	TrackRelayerStatus(unavailable peer.IDSlice, all peer.IDSlice)
}

func StartCommunicationHealthCheckJob(h host.Host, interval time.Duration, metrics Metrics) {
	healthComm := p2p.NewCommunication(h, "p2p/health")
	for {
		time.Sleep(interval)
		log.Info().Msg("Starting communication health check")

		all := h.Peerstore().Peers()
		unavailable := make(peer.IDSlice, 0)

		communicationErrors := comm.ExecuteCommHealthCheck(healthComm, h.Peerstore().Peers())
		for _, cerr := range communicationErrors {
			log.Err(cerr).Msg("communication error")
			unavailable = append(unavailable, cerr.Peer)
		}

		metrics.TrackRelayerStatus(unavailable, all)
	}
}
