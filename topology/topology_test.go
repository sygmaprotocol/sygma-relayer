package topology

import (
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type TopologyTestSuite struct {
	suite.Suite
}

func TestRunTopologyTestSuite(t *testing.T) {
	suite.Run(t, new(TopologyTestSuite))
}

func (s *TopologyTestSuite) SetupSuite()    {}
func (s *TopologyTestSuite) TearDownSuite() {}
func (s *TopologyTestSuite) SetupTest() {
	os.Clearenv()
}
func (s *TopologyTestSuite) TearDownTest() {}

func (s *TopologyTestSuite) Test_ProcessRawTopology_ValidTopology() {
	topology, err := ProcessRawTopology(&RawTopology{
		Peers: []RawPeer{
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
	_, err := ProcessRawTopology(&RawTopology{
		Peers: []RawPeer{
			{PeerAddress: "/dns4/relayer2/tcp/9001/p2p/"},
			{PeerAddress: "/dns4/relayer3/tcp/9002/p2p/QmYAYuLUPNwYEBYJaKHcE7NKjUhiUV8txx2xDXHvcYa1xK"},
			{PeerAddress: "/dns4/relayer1/tcp/9000/p2p/QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX"},
		},
		Threshold: "2",
	})
	s.NotNil(err)
}

func (s *TopologyTestSuite) Test_ProcessRawTopology_InvalidThreshold() {
	rt := &RawTopology{
		Peers: []RawPeer{
			{PeerAddress: "/dns4/relayer2/tcp/9001/p2p/QmeTuMtdpPB7zKDgmobEwSvxodrf5aFVSmBXX3SQJVjJaT"},
			{PeerAddress: "/dns4/relayer3/tcp/9002/p2p/QmYAYuLUPNwYEBYJaKHcE7NKjUhiUV8txx2xDXHvcYa1xK"},
			{PeerAddress: "/dns4/relayer1/tcp/9000/p2p/QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX"},
		},
		Threshold: "-1",
	}
	_, err := ProcessRawTopology(rt)
	s.NotNil(err)

	rt.Threshold = "0"
	_, err = ProcessRawTopology(rt)
	s.NotNil(err)

	rt.Threshold = "1"
	_, err = ProcessRawTopology(rt)
	s.NotNil(err)
}
