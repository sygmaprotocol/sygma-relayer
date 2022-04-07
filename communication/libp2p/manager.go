package libp2p

import (
	comm "github.com/ChainSafe/chainbridge-core/communication"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"sync"
)

// StreamManager manages instances of network.Stream
//
// Each stream is connected to a specific session, by sessionID
type StreamManager struct {
	unusedStreams map[comm.SessionID][]network.Stream
	streamLocker  *sync.RWMutex
	logger        zerolog.Logger
}

// NewStreamManager creates new StreamManager
func NewStreamManager() *StreamManager {
	return &StreamManager{
		unusedStreams: make(map[comm.SessionID][]network.Stream),
		streamLocker:  &sync.RWMutex{},
		logger:        log.With().Str("module", "communication").Logger(),
	}
}

// ReleaseStream removes reference on streams mapped to provided sessionID
func (sm *StreamManager) ReleaseStream(sessionID comm.SessionID) {
	sm.streamLocker.RLock()
	usedStreams, okStream := sm.unusedStreams[sessionID]
	unknownStreams, okUnknown := sm.unusedStreams["UNKNOWN"]
	sm.streamLocker.RUnlock()
	streamsForReset := append(usedStreams, unknownStreams...)
	if okStream || okUnknown {
		// close all streamsForReset
		for _, el := range streamsForReset {
			err := el.Reset()
			if err != nil {
				sm.logger.Error().Err(err).Msg("fail to reset the stream,skip it")
			}
		}
		sm.streamLocker.Lock()
		delete(sm.unusedStreams, sessionID)
		sm.streamLocker.Unlock()
	}
}

// AddStream saves and maps provided stream to provided sessionID
func (sm *StreamManager) AddStream(sessionID comm.SessionID, stream network.Stream) {
	if stream == nil {
		return
	}
	sm.streamLocker.Lock()
	defer sm.streamLocker.Unlock()
	entries, ok := sm.unusedStreams[sessionID]
	if !ok {
		entries := []network.Stream{stream}
		sm.unusedStreams[sessionID] = entries
	} else {
		entries = append(entries, stream)
		sm.unusedStreams[sessionID] = entries
	}
}
