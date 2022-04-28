package p2p

import (
	"github.com/ChainSafe/chainbridge-core/config/relayer"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/suite"
	"testing"
)

type HostTestSuite struct {
	suite.Suite
	mockController *gomock.Controller
}

func TestRunHostTestSuite(t *testing.T) {
	suite.Run(t, new(HostTestSuite))
}

func (s *HostTestSuite) SetupSuite()    {}
func (s *HostTestSuite) TearDownSuite() {}
func (s *HostTestSuite) SetupTest() {
	s.mockController = gomock.NewController(s.T())
}
func (s *HostTestSuite) TearDownTest() {}

func (s *HostTestSuite) TestHost_NewHost_Success() {
	p1RawAddress := "/ip4/127.0.0.1/tcp/4000/p2p/QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR"
	p2RawAddress := "/ip4/127.0.0.1/tcp/4002/p2p/QmeWhpY8tknHS29gzf9TAsNEwfejTCNJ7vFpmkV6rNUgyq"

	privKey, _, err := crypto.GenerateKeyPair(2, 0)
	if err != nil {
		return
	}
	s.Nil(err)

	p1, _ := peer.AddrInfoFromString(p1RawAddress)
	p2, _ := peer.AddrInfoFromString(p2RawAddress)

	host, err := NewHost(privKey, relayer.MpcRelayerConfig{
		Peers: []*peer.AddrInfo{
			p1, p2,
		},
		Port: 2020,
	})
	s.Nil(err)
	s.NotNil(host)
	// 2 peers + host
	s.Len(host.Peerstore().Peers(), 3)
}

func (s *HostTestSuite) TestHost_NewHost_InvalidPrivKey() {
	host, err := NewHost(nil, relayer.MpcRelayerConfig{
		Peers: []*peer.AddrInfo{},
		Port:  2020,
	})
	s.Nil(host)
	s.NotNil(err)
}
