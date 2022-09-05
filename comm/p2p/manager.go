// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package p2p

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/rs/zerolog/log"
	"sync"
)

// StreamManager manages instances of network.Stream
//
// Each stream is mapped to a specific session, by sessionID
type StreamManager struct {
	streamsBySessionID map[string][]network.Stream
	streamLocker       *sync.RWMutex
}

// NewStreamManager creates new StreamManager
func NewStreamManager() *StreamManager {
	return &StreamManager{
		streamsBySessionID: make(map[string][]network.Stream),
		streamLocker:       &sync.RWMutex{},
	}
}

// ReleaseStream removes reference on streams mapped to provided sessionID
func (sm *StreamManager) ReleaseStream(sessionID string) {
	sm.streamLocker.RLock()
	usedStreams, okStream := sm.streamsBySessionID[sessionID]
	unknownStreams, okUnknown := sm.streamsBySessionID["UNKNOWN"]
	sm.streamLocker.RUnlock()
	streamsForReset := append(usedStreams, unknownStreams...)
	if okStream || okUnknown {
		// close all streamsForReset
		for _, el := range streamsForReset {
			err := el.Reset()
			if err != nil {
				log.Error().Err(err).Msgf("failed to reset the stream %s, skip it", el.ID())
			}
		}
		sm.streamLocker.Lock()
		delete(sm.streamsBySessionID, sessionID)
		sm.streamLocker.Unlock()
	}
}

// AddStream saves and maps provided stream to provided sessionID
func (sm *StreamManager) AddStream(sessionID string, stream network.Stream) {
	if stream == nil {
		return
	}
	sm.streamLocker.Lock()
	defer sm.streamLocker.Unlock()
	entries, ok := sm.streamsBySessionID[sessionID]
	if !ok {
		entries := []network.Stream{stream}
		sm.streamsBySessionID[sessionID] = entries
	} else {
		entries = append(entries, stream)
		sm.streamsBySessionID[sessionID] = entries
	}
}
