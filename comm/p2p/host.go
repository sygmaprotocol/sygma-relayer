package p2p

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/sygma/topology"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	noise "github.com/libp2p/go-libp2p-noise"
	"github.com/rs/zerolog/log"
)

// NewHost creates new host.Host from private key and relayer configuration
func NewHost(privKey crypto.PrivKey, networkTopology topology.NetworkTopology, port uint16) (host.Host, error) {
	if privKey == nil {
		return nil, errors.New("unable to create libp2p host: private key not defined")
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
		libp2p.Identity(privKey),
		libp2p.DisableRelay(),
		libp2p.Security(noise.ID, noise.New),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create libp2p host: %v", err)
	}

	log.Info().Str("peerID", h.ID().Pretty()).Msgf(
		"new libp2p host created with address: %s", h.Addrs()[0].String(),
	)

	LoadPeers(h, networkTopology.Peers)
	return h, nil
}

// LoadPeers clears out peerstore and loads new peers into it
func LoadPeers(h host.Host, peers []*peer.AddrInfo) {
	for _, p := range h.Peerstore().Peers() {
		if p == h.ID() {
			continue
		}

		h.Peerstore().RemovePeer(p)
	}

	for _, p := range peers {
		h.Peerstore().AddAddr(p.ID, p.Addrs[0], peerstore.PermanentAddrTTL)
	}
}
