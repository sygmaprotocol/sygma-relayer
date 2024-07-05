// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package topology

import (
	"fmt"
	"net/http"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/comm/p2p"
	"github.com/ChainSafe/sygma-relayer/config/relayer"
	"github.com/ChainSafe/sygma-relayer/topology"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	pingCMD = &cobra.Command{
		Use:   "ping",
		Short: "Ping all nodes in the topology once",
		RunE:  runPing,
	}
)

var (
	pingEncryptionKey string
	pingURL           string
	pingPath          string
	pingP2PKey        string
)

func init() {
	pingCMD.PersistentFlags().StringVar(&pingEncryptionKey, "decryption-key", "", "Encryption key for the topology")
	_ = pingCMD.MarkFlagRequired("decryption-key")
	pingCMD.PersistentFlags().StringVar(&pingURL, "url", "", "URL for the topology")
	_ = pingCMD.MarkFlagRequired("url")
	pingCMD.PersistentFlags().StringVar(&pingP2PKey, "private-key", "", "P2P key")
	_ = pingCMD.MarkFlagRequired("private-key")
}

func runPing(cmd *cobra.Command, args []string) error {
	return runHealthCheck(pingEncryptionKey, pingURL, pingPath, pingP2PKey)
}

func runHealthCheck(encryptionKey, url, path, p2pKey string) error {
	topologyProvider, err := topology.NewNetworkTopologyProvider(relayer.TopologyConfiguration{
		EncryptionKey: encryptionKey,
		Url:           url,
		Path:          path,
	}, http.DefaultClient)
	if err != nil {
		return fmt.Errorf("failed to create topology provider: %w", err)
	}

	networkTopology, err := topologyProvider.NetworkTopology("")
	if err != nil {
		return fmt.Errorf("failed to get network topology: %w", err)
	}

	privBytes, err := crypto.ConfigDecodeKey(p2pKey)
	if err != nil {
		return fmt.Errorf("failed to decode P2P key: %w", err)
	}

	priv, err := crypto.UnmarshalPrivateKey(privBytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal private key: %w", err)
	}

	connectionGate := p2p.NewConnectionGate(networkTopology)
	host, err := p2p.NewHost(priv, networkTopology, connectionGate, 9000)
	if err != nil {
		return fmt.Errorf("failed to create host: %w", err)
	}

	err = executeCommHealthCheck(host)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

func executeCommHealthCheck(h host.Host) error {
	log.Debug().Msg("Starting communication health check")

	all := h.Peerstore().Peers()
	unavailable := make(peer.IDSlice, 0)

	healthComm := p2p.NewCommunication(h, "p2p/health")
	communicationErrors := comm.ExecuteCommHealthCheck(healthComm, h.Peerstore().Peers())
	for _, cerr := range communicationErrors {
		log.Err(cerr).Msg("communication error on ExecuteCommHealthCheck")
		unavailable = append(unavailable, cerr.Peer)
	}

	trackRelayerStatus(unavailable, all)
	return nil
}

func trackRelayerStatus(unavailable peer.IDSlice, all peer.IDSlice) {
	fmt.Printf("Unavailable peers: %v\n", unavailable)
	fmt.Printf("All peers: %v\n", all)
}
