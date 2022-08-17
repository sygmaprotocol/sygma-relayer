package topology

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ChainSafe/sygma/config/relayer"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/rs/zerolog/log"
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

	return strconv.FormatUint(hash, 10), nil
}

func (nt NetworkTopology) PeerIDS() peer.IDSlice {
	peerIDS := make(peer.IDSlice, len(nt.Peers))
	for i, peer := range nt.Peers {
		peerIDS[i] = peer.ID
	}
	return peerIDS
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
	client, err := minio.New(config.ServiceAddress, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecKey, ""),
		Secure: true,
		Region: config.BucketRegion,
	})
	if err != nil {
		return nil, err
	}

	return &topologyProvider{
		client:       *client,
		documentName: config.DocumentName,
		bucketName:   config.BucketName,
	}, nil
}

type RawTopology struct {
	Peers     []RawPeer `mapstructure:"Peers" json:"peers"`
	Threshold string    `mapstructure:"Threshold" json:"threshold"`
}

type RawPeer struct {
	PeerAddress string `mapstructure:"PeerAddress" json:"peerAddress"`
}

type topologyProvider struct {
	client       minio.Client
	documentName string
	bucketName   string
}

func (t *topologyProvider) NetworkTopology() (NetworkTopology, error) {
	ctx := context.Background()

	obj, err := t.client.GetObject(ctx, t.bucketName, t.documentName, minio.GetObjectOptions{})
	if err != nil {
		log.Err(err).Msg("unable to get topology object")
		return NetworkTopology{}, err
	}

	stat, err := obj.Stat()
	if err != nil {
		log.Err(err).Msg("unable to get topology object information")
		return NetworkTopology{}, err
	}

	data := make([]byte, stat.Size)
	_, err = obj.Read(data)
	if err != nil {
		log.Err(err).Msg("error on reading topology data")
	}

	rawTopology := &RawTopology{}
	err = json.Unmarshal(data, rawTopology)
	if err != nil {
		log.Err(err).Msg("unable to unmarshal topology data")
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
