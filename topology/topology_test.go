// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package topology_test

import (
	"os"
	"testing"

	"github.com/ChainSafe/sygma-relayer/topology"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

type TopologyTestSuite struct {
	suite.Suite
}

func TestRunTopologyTestSuite(t *testing.T) {
	suite.Run(t, new(TopologyTestSuite))
}

func (s *TopologyTestSuite) SetupTest() {
	os.Clearenv()
}

func (s *TopologyTestSuite) Test_ProcessRawTopology_ValidTopology() {
	topology, err := topology.ProcessRawTopology(&topology.RawTopology{
		Peers: []topology.RawPeer{
			{PeerAddress: "/dns4/relayer2/tcp/9001/p2p/QmeTuMtdpPB7zKDgmobEwSvxodrf5aFVSmBXX3SQJVjJaT"},
			{PeerAddress: "/dns4/relayer3/tcp/9002/p2p/QmYAYuLUPNwYEBYJaKHcE7NKjUhiUV8txx2xDXHvcYa1xK"},
			{PeerAddress: "/dns4/relayer1/tcp/9000/p2p/QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX"},
		},
		Threshold: "2",
	})
	s.Nil(err)
	s.Equal(2, topology.Threshold)
	s.Equal("QmeTuMtdpPB7zKDgmobEwSvxodrf5aFVSmBXX3SQJVjJaT", topology.Peers[0].ID.Pretty())
	s.Equal("QmYAYuLUPNwYEBYJaKHcE7NKjUhiUV8txx2xDXHvcYa1xK", topology.Peers[1].ID.Pretty())
	s.Equal("QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX", topology.Peers[2].ID.Pretty())
}

func (s *TopologyTestSuite) Test_ProcessRawTopology_InvalidPeerAddress() {
	_, err := topology.ProcessRawTopology(&topology.RawTopology{
		Peers: []topology.RawPeer{
			{PeerAddress: "/dns4/relayer2/tcp/9001/p2p/"},
			{PeerAddress: "/dns4/relayer3/tcp/9002/p2p/QmYAYuLUPNwYEBYJaKHcE7NKjUhiUV8txx2xDXHvcYa1xK"},
			{PeerAddress: "/dns4/relayer1/tcp/9000/p2p/QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX"},
		},
		Threshold: "2",
	})
	s.NotNil(err)
}

func (s *TopologyTestSuite) Test_ProcessRawTopology_InvalidThreshold() {
	rt := &topology.RawTopology{
		Peers: []topology.RawPeer{
			{PeerAddress: "/dns4/relayer2/tcp/9001/p2p/QmeTuMtdpPB7zKDgmobEwSvxodrf5aFVSmBXX3SQJVjJaT"},
			{PeerAddress: "/dns4/relayer3/tcp/9002/p2p/QmYAYuLUPNwYEBYJaKHcE7NKjUhiUV8txx2xDXHvcYa1xK"},
			{PeerAddress: "/dns4/relayer1/tcp/9000/p2p/QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX"},
		},
		Threshold: "-1",
	}
	_, err := topology.ProcessRawTopology(rt)
	s.NotNil(err)

	rt.Threshold = "0"
	_, err = topology.ProcessRawTopology(rt)
	s.NotNil(err)

	rt.Threshold = "1"
	_, err = topology.ProcessRawTopology(rt)
	s.Nil(err)
}

type NetworkTopologyTestSuite struct {
	suite.Suite
}

func TestRunNetworkTopologyTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkTopologyTestSuite))
}

func (s *NetworkTopologyTestSuite) Test_Hash_ValidHash() {
	p1RawAddress := "/ip4/127.0.0.1/tcp/4000/p2p/QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR"
	p2RawAddress := "/ip4/127.0.0.1/tcp/4002/p2p/QmeWhpY8tknHS29gzf9TAsNEwfejTCNJ7vFpmkV6rNUgyq"
	p1, _ := peer.AddrInfoFromString(p1RawAddress)
	p2, _ := peer.AddrInfoFromString(p2RawAddress)
	topology := topology.NetworkTopology{
		Peers: []*peer.AddrInfo{
			p1, p2,
		},
		Threshold: 2,
	}

	hash, _ := topology.Hash()

	s.Equal(hash, "8c6541d0e584f0c0")
}

func (s *NetworkTopologyTestSuite) Test_IsAllowedPeer_ValidPeer() {
	p1RawAddress := "/ip4/127.0.0.1/tcp/4000/p2p/QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR"
	p2RawAddress := "/ip4/127.0.0.1/tcp/4002/p2p/QmeWhpY8tknHS29gzf9TAsNEwfejTCNJ7vFpmkV6rNUgyq"
	p1, _ := peer.AddrInfoFromString(p1RawAddress)
	p2, _ := peer.AddrInfoFromString(p2RawAddress)
	topology := topology.NetworkTopology{
		Peers: []*peer.AddrInfo{
			p1, p2,
		},
		Threshold: 2,
	}

	isAllowed := topology.IsAllowedPeer(p1.ID)

	s.Equal(isAllowed, true)
}

func (s *NetworkTopologyTestSuite) Test_IsAllowedPeer_InvalidPeer() {
	p1RawAddress := "/ip4/127.0.0.1/tcp/4000/p2p/QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR"
	p2RawAddress := "/ip4/127.0.0.1/tcp/4002/p2p/QmeWhpY8tknHS29gzf9TAsNEwfejTCNJ7vFpmkV6rNUgyq"
	p3RawAddress := "/ip4/127.0.0.1/tcp/4002/p2p/QmeWhpY8tknHS29gzf9TAsNEwfejTCNJ7vFpmkV6rNUgyg"
	p1, _ := peer.AddrInfoFromString(p1RawAddress)
	p2, _ := peer.AddrInfoFromString(p2RawAddress)
	p3, _ := peer.AddrInfoFromString(p3RawAddress)
	topology := topology.NetworkTopology{
		Peers: []*peer.AddrInfo{
			p1, p2,
		},
		Threshold: 2,
	}

	isAllowed := topology.IsAllowedPeer(p3.ID)

	s.Equal(isAllowed, false)
}
