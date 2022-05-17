package bully

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"strings"
	"sync"
)

type Peers interface {
	Add(ID peer.ID)
	Delete(ID peer.ID)
	Find(ID peer.ID) bool
	PeerData() []peer.ID
}

//  0 if a == b
//  1 if a > b
// -1 if a < b
func ComparePeerIDs(a peer.ID, b peer.ID) int {
	return strings.Compare(a.Pretty(), b.Pretty())
}

type PeerMap struct {
	mu    *sync.RWMutex
	peers map[string]*peer.ID
}

func NewPeerMap() *PeerMap {
	return &PeerMap{mu: &sync.RWMutex{}, peers: make(map[string]*peer.ID)}
}

// NOTE: This function is thread-safe.
func (pm *PeerMap) Add(ID peer.ID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.peers[ID.Pretty()] = &ID
}

// NOTE: This function is thread-safe.
func (pm *PeerMap) Delete(ID peer.ID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.peers, ID.Pretty())
}

// NOTE: This function is thread-safe.
func (pm *PeerMap) Find(ID peer.ID) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	_, ok := pm.peers[ID.Pretty()]
	return ok
}

// NOTE: This function is thread-safe.
func (pm *PeerMap) PeerData() []peer.ID {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var IDSlice []peer.ID
	for _, p := range pm.peers {
		IDSlice = append(IDSlice, *p)
	}
	return IDSlice
}
