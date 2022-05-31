package health

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

// StartHealthEndpoint starts /health endpoint on provided port that returns ok on invocation
func StartHealthEndpoint(port string) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	_ = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	log.Info().Msgf("started /health endpoint on port %s", port)
}
