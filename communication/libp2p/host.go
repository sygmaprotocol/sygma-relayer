package libp2p

import (
	"fmt"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

func NewHost(privKey crypto.PrivKey, rconf config.RelayerConfig) (host.Host, error) {
	logger := log.With().Str("relayer", rconf.Name).Logger()
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", rconf.Port)),
		libp2p.Identity(privKey),
		libp2p.DisableRelay(),
		libp2p.Security(noise.ID, noise.New),
	}

	h, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		logger.Error().Msg(
			"unable to create libp2p host",
		)
		return nil, err
	}

	fullAddr := util.GetHostAddress(h)
	logger.Info().Msg(
		fmt.Sprintf("host %s created", fullAddr),
	)

	for _, p := range rconf.Peers {
		ma, err := multiaddr.NewMultiaddr(p.PeerAddress)
		if err != nil {
			logger.Error().Msg(
				fmt.Sprintf("unable to create multiaddr: %s", err.Error()),
			)
		}

		addrInfo, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			logger.Error().Msg(
				fmt.Sprintf("unable to parse peer address: %s", p.PeerAddress),
			)
			return nil, err
		}
		h.Peerstore().AddAddr(addrInfo.ID, addrInfo.Addrs[0], peerstore.PermanentAddrTTL)
	}
	return h, nil
}
