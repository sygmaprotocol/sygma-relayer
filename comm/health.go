// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package comm

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

const HealthTimeout = 10 * time.Second

func ExecuteCommHealthCheck(communication Communication, peers peer.IDSlice) []*CommunicationError {
	sessionID := "health-session"
	defer communication.CloseSession(sessionID)

	errors := make([]*CommunicationError, 0)
	for _, p := range peers {
		err := communication.Broadcast([]peer.ID{p}, []byte{}, Unknown, sessionID)
		if err != nil {
			errors = append(errors, err.(*CommunicationError))
		}
	}
	return errors
}
