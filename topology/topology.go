// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package topology

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/ChainSafe/sygma-relayer/config/relayer"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mitchellh/hashstructure/v2"
)

type NetworkTopology struct {
	Peers     []*peer.AddrInfo
	Threshold int
}

func (nt NetworkTopology) Hash() (string, error) {
	hash, err := hashstructure.Hash(nt, hashstructure.FormatV2, nil)
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(hash, 16), nil
}

func (nt NetworkTopology) IsAllowedPeer(peer peer.ID) bool {
	for _, p := range nt.Peers {
		if p.ID == peer {
			return true
		}
	}

	return false
}

type NetworkTopologyProvider interface {
	NetworkTopology() (NetworkTopology, error)
}

func NewNetworkTopologyProvider(config relayer.TopologyConfiguration) (NetworkTopologyProvider, error) {
	decrypter, err := NewAESEncryption([]byte(config.EncryptionKey))
	if err != nil {
		return nil, err
	}

	return &topologyProvider{
		decrypter: decrypter,
		url:       config.Url,
	}, nil
}

type RawTopology struct {
	Peers     []RawPeer `mapstructure:"Peers" json:"peers"`
	Threshold string    `mapstructure:"Threshold" json:"threshold"`
}

type RawPeer struct {
	PeerAddress string `mapstructure:"PeerAddress" json:"peerAddress"`
}

type Decrypter interface {
	Decrypt(data string) []byte
}

type topologyProvider struct {
	url       string
	decrypter Decrypter
}

func (t *topologyProvider) NetworkTopology() (NetworkTopology, error) {
	resp, err := http.Get(t.url)
	if err != nil {
		return NetworkTopology{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return NetworkTopology{}, err
	}

	rawTopology := &RawTopology{}
	err = json.Unmarshal(body, rawTopology)
	if err != nil {
		return NetworkTopology{}, err
	}

	return ProcessRawTopology(rawTopology)
}

func ProcessRawTopology(rawTopology *RawTopology) (NetworkTopology, error) {
	var peers []*peer.AddrInfo
	for _, p := range rawTopology.Peers {
		addrInfo, err := peer.AddrInfoFromString(p.PeerAddress)
		if err != nil {
			return NetworkTopology{}, fmt.Errorf("invalid peer address %s: %w", p.PeerAddress, err)
		}
		peers = append(peers, addrInfo)
	}

	threshold, err := strconv.ParseInt(rawTopology.Threshold, 0, 0)
	if err != nil {
		return NetworkTopology{}, fmt.Errorf("unable to parse mpc threshold from topology %v", err)
	}
	if threshold <= 1 {
		return NetworkTopology{}, fmt.Errorf("mpc threshold must be bigger then 1 %v", err)
	}
	return NetworkTopology{Peers: peers, Threshold: int(threshold)}, nil
}
