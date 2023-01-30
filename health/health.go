// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package health

import (
	"fmt"
	"net/http"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

// StartHealthEndpoint starts /health endpoint on provided port that returns ok on invocation
func StartHealthEndpoint(port uint16, c comm.Communication, h host.Host) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Msg("Health endpoint called")
		go comm.ExecuteCommHealthCheck(c, h.Peerstore().Peers())
		_, _ = w.Write([]byte("ok"))
	})
	_ = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	log.Info().Msgf("started /health endpoint on port %d", port)
}
