package config_test

import (
	"encoding/json"
	"github.com/libp2p/go-libp2p-core/peer"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ChainSafe/chainbridge-core/config"
	"github.com/ChainSafe/chainbridge-core/config/relayer"
	"github.com/stretchr/testify/suite"
)

type GetConfigTestSuite struct {
	suite.Suite
}

func TestRunGetConfigTestSuite(t *testing.T) {
	suite.Run(t, new(GetConfigTestSuite))
}

func (s *GetConfigTestSuite) SetupSuite()    {}
func (s *GetConfigTestSuite) TearDownSuite() {}
func (s *GetConfigTestSuite) SetupTest()     {}
func (s *GetConfigTestSuite) TearDownTest()  {}

func (s *GetConfigTestSuite) Test_InvalidPath() {
	_, err := config.GetConfig("invalid")

	s.NotNil(err)
}

func (s *GetConfigTestSuite) Test_MissingChainType() {
	data := config.RawConfig{
		ChainConfigs: []map[string]interface{}{{
			"name": "chain1",
		}},
	}
	file, _ := json.Marshal(data)
	_ = ioutil.WriteFile("test.json", file, 0644)

	_, err := config.GetConfig("test.json")

	_ = os.Remove("test.json")
	s.NotNil(err)
	s.Equal(err.Error(), "chain 'type' must be provided for every configured chain")
}

func (s *GetConfigTestSuite) Test_InvalidRelayerConfig() {
	data := config.RawConfig{
		RelayerConfig: relayer.RawRelayerConfig{
			LogLevel: "invalid",
		},
		ChainConfigs: []map[string]interface{}{{
			"name": "chain1",
		}},
	}
	file, _ := json.Marshal(data)
	_ = ioutil.WriteFile("test.json", file, 0644)

	_, err := config.GetConfig("test.json")

	_ = os.Remove("test.json")
	s.NotNil(err)
	s.Equal(err.Error(), "unknown log level: invalid")
}

func (s *GetConfigTestSuite) Test_InvalidPeerAddress() {
	data := config.RawConfig{
		RelayerConfig: relayer.RawRelayerConfig{
			LogLevel: "info",
			MpcConfig: relayer.RawMpcRelayerConfig{
				Peers: []relayer.RawPeer{
					{PeerAddress: "/ip4/127.0.0.1/tcp/4000"},
				},
				Port: 2020,
			},
		},
		ChainConfigs: []map[string]interface{}{{
			"name": "chain1",
		}},
	}
	file, _ := json.Marshal(data)
	_ = ioutil.WriteFile("test.json", file, 0644)

	_, err := config.GetConfig("test.json")

	_ = os.Remove("test.json")
	s.NotNil(err)
	s.Equal(err.Error(), "invalid peer address /ip4/127.0.0.1/tcp/4000: invalid p2p multiaddr")
}

func (s *GetConfigTestSuite) Test_ValidConfig() {
	p1RawAddress := "/ip4/127.0.0.1/tcp/4000/p2p/QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR"
	p2RawAddress := "/ip4/127.0.0.1/tcp/4002/p2p/QmeWhpY8tknHS29gzf9TAsNEwfejTCNJ7vFpmkV6rNUgyq"
	data := config.RawConfig{
		RelayerConfig: relayer.RawRelayerConfig{
			LogLevel: "info",
			MpcConfig: relayer.RawMpcRelayerConfig{
				Peers: []relayer.RawPeer{
					{PeerAddress: p1RawAddress},
					{PeerAddress: p2RawAddress},
				},
				Port: 2020,
			},
		},
		ChainConfigs: []map[string]interface{}{{
			"type": "evm",
			"name": "evm1",
		}},
	}
	file, _ := json.Marshal(data)
	_ = ioutil.WriteFile("test.json", file, 0644)

	actualConfig, err := config.GetConfig("test.json")

	_ = os.Remove("test.json")

	p1, _ := peer.AddrInfoFromString(p1RawAddress)
	p2, _ := peer.AddrInfoFromString(p2RawAddress)

	s.Nil(err)
	s.Equal(actualConfig, config.Config{
		RelayerConfig: relayer.RelayerConfig{
			LogLevel:                  1,
			LogFile:                   "",
			OpenTelemetryCollectorURL: "",
			MpcConfig: relayer.MpcRelayerConfig{
				Peers: []*peer.AddrInfo{p1, p2},
				Port:  2020,
			},
		},
		ChainConfigs: []map[string]interface{}{{
			"type": "evm",
			"name": "evm1",
		}},
	})

}
