package p2p

import (
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"
	noise "github.com/libp2p/go-libp2p-noise"
	"github.com/rs/zerolog/log"
)

func NewHost(privKey crypto.PrivKey, rconf config.RelayerConfig) (host.Host, error) {
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", rconf.Port)),
		libp2p.Identity(privKey),
		libp2p.DisableRelay(),
		libp2p.Security(noise.ID, noise.New),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		log.Error().Msg(
			"unable to create libp2p host",
		)
		return nil, err
	}
	h.Addrs()[0].String()
	fullAddr := util.GetHostAddress(h)
	log.Info().Str("peerID", h.ID().Pretty()).Msg(
		fmt.Sprintf("host %s created", fullAddr),
	)

	for _, p := range rconf.Peers {
		h.Peerstore().AddAddr(addrInfo.ID, addrInfo.Addrs[0], peerstore.PermanentAddrTTL)
	}
	return h, nil
}
