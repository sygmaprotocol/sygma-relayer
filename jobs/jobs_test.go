package jobs_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ChainSafe/sygma-relayer/comm/p2p"
	"github.com/ChainSafe/sygma-relayer/config/relayer"
	"github.com/ChainSafe/sygma-relayer/jobs"
	"github.com/ChainSafe/sygma-relayer/topology"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

const (
	TopologyEncryptionKey = "fXLzaRcCHFYgg47j"
	TopologyURL           = "https://sygma-assets-mainnet.s3.us-east-2.amazonaws.com/topology-mainnet"
	// Key for entity (peer) that will call ping to other relayers
	// Qmew8dKTTCM8XxvouVCaaSC36UGDiyQzrcH2jFg5TbRXut
	P2PKey = "CAASqAkwggSkAgEAAoIBAQCcrxN8Vw+wFgrRhHfB11OyGtxqSSZXxY97nRoZQs6CuhnYmTj6r4s/xn3eni8h29xKypXgCEFRcUtGJ9MTom87B5vpJuxFEc9p2fML41ff8A75zlxOX6b9O2woJPhIKrjzc+oDL1RHJYaOgIMe6OlPG/HA9IanOaoL6P0MKpCJAMC92VfSsPjsj0/kxpQghjEZ3d9bJ2HzhXahUFLMKrDL08MCOoWg8NIL45SYMros9+dWWoHPlp6aXZftgAMq7F/HyU4a5+RmayTV9iH5YDd1JDlgnQrfjKN+KVNAtkYjvStcxrrJwvZ+/E3LXj2Gs573cxH7yAyf2jE6DYsECcPrAgMBAAECggEBAIh+bDcxkNURHrOO9tTCxIKvq7xbpS9pR6mkHoxLCqQPg1yRfnXEX0ZboGNC4kRYx/vPp+jWyDAuxiiDoPnF06hU5Jmj8sfo0AuidnywvGi1NBaikv8jjNGl5n7CVhhoP162Z/IGVSD1q9aQVamjtTvZWC2D15nuPhjKC0eB/Q+bXzlWS7zMUNnELX8mlbbx7XE6oIR/iXhv2XMn+e/9G+IwFwkBTH+2R1RegLvr7MMNkTF+RJyQ5PUBn5ZF2GGCRJT971Ee3ATHY86nu+BtbGUgGaO1ftclUt8eElKL2KZOFMCqv2ZBgbAbivaUOJiSy9ZJImX8DaXZK7Duft3Mz8ECgYEAzdnl83flrb05wc1cEfs4BhRjE8J/oqc1+8dZfIbaERAo9MrQARYjhAvxolyYzI+qdoyzwSMfbOa67sBSz29vv98Pkvs4vcupJ/YWbzPzjqRuOm5PIGAqnFt8vL8OK/ynCZbBzFRtRNFjVmuYVehflCii0q65J8TOJmRNXiuarAMCgYEAwtrQhQm9wziRDUvcRbZtRGwpqcFMQ9hQC532Rqz0BWpKt5ONN5MO2NSIJ6vOQyNBeedEHQNk+3kG65NsToyuFSjoM58uYidOftjmFxGAww/lqENKAzFJ/BElgXb+Msl7/pa5cE9p/3DzVPRXHjZEPSGelFt3uze+0jOiuajMJ/kCgYEArNVDhbzoIYyb3sU+hXZo3mnlmeSW54j/AUuqLazHkMYBrS5PoGnnHeotUgXu4OnK1Mhj8Eg+DWBYGTdvD+1fZTiyydSWGnzRpNSwl2OGHgCe7/5H/0Xe4PLLc2nySypRUPK7+oP0TnCDuD6UY6S8AxhvRPcgTGyoLYHPl76CmeMCgYAn+wvD8F6+WrHwf3s/1pGO836M9Tt3xD+QUqYAlGYxYkDYb+8O0x69wMX7FdZpkidSIvCn31Vt/8Q6u/ICH/1sHAug4+15eEUz4786Rn4cB/wATWY3R3q9vKrsaIT52LuXXkfIUpMWNY/IA6aIbWwM+wP1vtrPUD3YFX4zB/5zyQKBgBhuf1M4WW1sEFuqXcO8EM6h222LewIsQz4wh3Xj67MGBut9sBO8zLBrM0ZMGoZUfOOgbvEKFbm+/Icy9U2WmbRSDPDMv96TNusHMvVrHjlp5MKgrSzxnM/2jYYaqRRzf+1LDg3hYmOK3t3tZ5xnBz49JNodSOrcN7wFg1tagN0Z"
)

type JobsTestSuite struct {
	suite.Suite
}

func TestRunJobsTestSuite(t *testing.T) {
	suite.Run(t, new(JobsTestSuite))
}

func (s *JobsTestSuite) Test_PingEveryoneFromTopology() {
	topologyProvider, err := topology.NewNetworkTopologyProvider(relayer.TopologyConfiguration{
		EncryptionKey: TopologyEncryptionKey,
		Url:           TopologyURL,
		Path:          "",
	}, http.DefaultClient)
	if err != nil {
		fmt.Println(err)
		return
	}

	networkTopology, err := topologyProvider.NetworkTopology("")
	if err != nil {
		fmt.Println(err)
		return
	}

	privBytes, err := crypto.ConfigDecodeKey(P2PKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	priv, err := crypto.UnmarshalPrivateKey(privBytes)
	if err != nil {
		fmt.Println(err)
		return
	}

	connectionGate := p2p.NewConnectionGate(networkTopology)
	host, err := p2p.NewHost(priv, networkTopology, connectionGate, 9000)
	if err != nil {
		fmt.Println(err)
		return
	}

	jobs.StartCommunicationHealthCheckJob(host, time.Second*10, &relayerStatusMeterImpl{})
}

type relayerStatusMeterImpl struct{}

func (r *relayerStatusMeterImpl) TrackRelayerStatus(unavailable peer.IDSlice, all peer.IDSlice) {
	fmt.Printf("Unavailable peers: %v\n", unavailable)
	fmt.Printf("All peers: %v\n", all)
}
