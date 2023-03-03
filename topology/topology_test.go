// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package topology_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/ChainSafe/sygma-relayer/config/relayer"
	"github.com/ChainSafe/sygma-relayer/topology"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"

	mock_topology "github.com/ChainSafe/sygma-relayer/topology/mock"
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

type TopologyProviderTestSuite struct {
	suite.Suite
	fetcher *mock_topology.MockFetcher
}

func TestRunTopologyProviderTestSuite(t *testing.T) {
	suite.Run(t, new(TopologyProviderTestSuite))
}

func (s *TopologyProviderTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.fetcher = mock_topology.NewMockFetcher(ctrl)

}

func (s *TopologyProviderTestSuite) Test_FetchingTopologyFails() {
	s.fetcher.EXPECT().Get("test.url").Return(&http.Response{}, fmt.Errorf("error"))
	topologyConfiguration := relayer.TopologyConfiguration{
		Url:           "test.url",
		EncryptionKey: "v8y/B?E(H+MbQeTh",
	}
	topologyProvider, _ := topology.NewNetworkTopologyProvider(topologyConfiguration, s.fetcher)

	_, err := topologyProvider.NetworkTopology("")

	s.NotNil(err)
}

func (s *TopologyProviderTestSuite) Test_ValidTopology() {
	resp := &http.Response{}
	resp.Body = io.NopCloser(strings.NewReader("12345678123456786c49ea9bdd0d37f3ad3d266b4ef5d6ef243027b25bd5092091bcd26488e185c4f466c6795535593dc41b03fcd8997985a78bc784c9f561bac89683a0170e5632ec0fc9237a97ebe38f783067d7f0d19dfe708349ca10759e6091228de7899ee10d679c8b444132bfd8106e1d28e944facb21be60182b7c069f264244ab545871ee6d15a1f070cacada34647bc7d2404384b3ee54b9058ec14ae9e017610f392adeb05d33d524e10043908887d932e5a974c8200639c0dc8d77e1cfb65ecbd2f9c731c61212d1a928b5436f3540cfbd981070b5567ced664ef20cc795ebb792231df08f05987a5d9458664d34666995fb15a969440dfd28db35fbd79f9e11cbcfd42409259c4bb1006c0907d2d4b170698e90452ead9ab7f4e41309fe8c586ccee54cc9cfaf3ac22d00b6c6f583a3f7a1fe3ddd470aa12ad9cf63f072798dadb5a21004529fec4a5914d68a18fd0b3fc33079d4ff09af44416b732f024b75b40dd0"))
	s.fetcher.EXPECT().Get("test.url").Return(resp, nil)
	topologyConfiguration := relayer.TopologyConfiguration{
		Url:           "test.url",
		EncryptionKey: "v8y/B?E(H+MbQeTh",
	}
	topologyProvider, _ := topology.NewNetworkTopologyProvider(topologyConfiguration, s.fetcher)

	tp, err := topologyProvider.NetworkTopology("")

	rawTp, _ := topology.ProcessRawTopology(&topology.RawTopology{
		Peers: []topology.RawPeer{
			{PeerAddress: "/dns4/relayer2/tcp/9001/p2p/QmeTuMtdpPB7zKDgmobEwSvxodrf5aFVSmBXX3SQJVjJaT"},
			{PeerAddress: "/dns4/relayer3/tcp/9002/p2p/QmYAYuLUPNwYEBYJaKHcE7NKjUhiUV8txx2xDXHvcYa1xK"},
			{PeerAddress: "/dns4/relayer1/tcp/9000/p2p/QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX"},
		},
		Threshold: "2",
	})
	s.Nil(err)
	s.Equal(rawTp, tp)
}

func (s *TopologyProviderTestSuite) Test_InvalidHash() {
	resp := &http.Response{}
	resp.Body = io.NopCloser(strings.NewReader("12345678123456786c49ea9bdd0d37f3ad3d266b4ef5d6ef243027b25bd5092091bcd26488e185c4f466c6795535593dc41b03fcd8997985a78bc784c9f561bac89683a0170e5632ec0fc9237a97ebe38f783067d7f0d19dfe708349ca10759e6091228de7899ee10d679c8b444132bfd8106e1d28e944facb21be60182b7c069f264244ab545871ee6d15a1f070cacada34647bc7d2404384b3ee54b9058ec14ae9e017610f392adeb05d33d524e10043908887d932e5a974c8200639c0dc8d77e1cfb65ecbd2f9c731c61212d1a928b5436f3540cfbd981070b5567ced664ef20cc795ebb792231df08f05987a5d9458664d34666995fb15a969440dfd28db35fbd79f9e11cbcfd42409259c4bb1006c0907d2d4b170698e90452ead9ab7f4e41309fe8c586ccee54cc9cfaf3ac22d00b6c6f583a3f7a1fe3ddd470aa12ad9cf63f072798dadb5a21004529fec4a5914d68a18fd0b3fc33079d4ff09af44416b732f024b75b40dd0"))
	s.fetcher.EXPECT().Get("test.url").Return(resp, nil)
	topologyConfiguration := relayer.TopologyConfiguration{
		Url:           "test.url",
		EncryptionKey: "v8y/B?E(H+MbQeTh",
	}
	topologyProvider, _ := topology.NewNetworkTopologyProvider(topologyConfiguration, s.fetcher)

	_, err := topologyProvider.NetworkTopology("invalid")

	s.NotNil(err)
}

func (s *TopologyProviderTestSuite) Test_ValidHash() {
	resp := &http.Response{}
	resp.Body = io.NopCloser(strings.NewReader("12345678123456786c49ea9bdd0d37f3ad3d266b4ef5d6ef243027b25bd5092091bcd26488e185c4f466c6795535593dc41b03fcd8997985a78bc784c9f561bac89683a0170e5632ec0fc9237a97ebe38f783067d7f0d19dfe708349ca10759e6091228de7899ee10d679c8b444132bfd8106e1d28e944facb21be60182b7c069f264244ab545871ee6d15a1f070cacada34647bc7d2404384b3ee54b9058ec14ae9e017610f392adeb05d33d524e10043908887d932e5a974c8200639c0dc8d77e1cfb65ecbd2f9c731c61212d1a928b5436f3540cfbd981070b5567ced664ef20cc795ebb792231df08f05987a5d9458664d34666995fb15a969440dfd28db35fbd79f9e11cbcfd42409259c4bb1006c0907d2d4b170698e90452ead9ab7f4e41309fe8c586ccee54cc9cfaf3ac22d00b6c6f583a3f7a1fe3ddd470aa12ad9cf63f072798dadb5a21004529fec4a5914d68a18fd0b3fc33079d4ff09af44416b732f024b75b40dd0"))
	s.fetcher.EXPECT().Get("test.url").Return(resp, nil)
	topologyConfiguration := relayer.TopologyConfiguration{
		Url:           "test.url",
		EncryptionKey: "v8y/B?E(H+MbQeTh",
	}
	topologyProvider, _ := topology.NewNetworkTopologyProvider(topologyConfiguration, s.fetcher)

	expectedHash := "f5909a83374428a4d34ec475ff09f041722145ee7e935847eec9f0b483f9ff06"
	tp, err := topologyProvider.NetworkTopology(expectedHash)

	rawTp, _ := topology.ProcessRawTopology(&topology.RawTopology{
		Peers: []topology.RawPeer{
			{PeerAddress: "/dns4/relayer2/tcp/9001/p2p/QmeTuMtdpPB7zKDgmobEwSvxodrf5aFVSmBXX3SQJVjJaT"},
			{PeerAddress: "/dns4/relayer3/tcp/9002/p2p/QmYAYuLUPNwYEBYJaKHcE7NKjUhiUV8txx2xDXHvcYa1xK"},
			{PeerAddress: "/dns4/relayer1/tcp/9000/p2p/QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX"},
		},
		Threshold: "2",
	})
	s.Nil(err)
	s.Equal(rawTp, tp)
}
