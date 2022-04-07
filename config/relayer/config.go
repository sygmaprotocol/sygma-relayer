package relayer

import (
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog"
)

type RelayerConfig struct {
	OpenTelemetryCollectorURL string
	LogLevel                  zerolog.Level
	LogFile                   string
	MpcConfig                 MpcRelayerConfig
}

type MpcRelayerConfig struct {
	Peers []*peer.AddrInfo
	Port  int64
}

type RawRelayerConfig struct {
	OpenTelemetryCollectorURL string              `mapstructure:"OpenTelemetryCollectorURL" json:"opentelemetryCollectorURL"`
	LogLevel                  string              `mapstructure:"LogLevel" json:"logLevel"`
	LogFile                   string              `mapstructure:"LogFile" json:"logFile"`
	MpcConfig                 RawMpcRelayerConfig `mapstructure:"MpcConfig" json:"mpcConfig"`
}

type RawMpcRelayerConfig struct {
	Peers []RawPeer `mapstructure:"Peers" json:"peers"`
	Port  int64     `mapstructure:"Port" json:"port"`
}

type RawPeer struct {
	PeerAddress string `mapstructure:"PeerAddress" json:"peerAddress"`
}

func (c *RawRelayerConfig) Validate() error {
	return nil
}

// NewRelayerConfig parses RawRelayerConfig into RelayerConfig
func NewRelayerConfig(rawConfig RawRelayerConfig) (RelayerConfig, error) {
	config := RelayerConfig{}
	err := rawConfig.Validate()
	if err != nil {
		return config, err
	}

	logLevel, err := zerolog.ParseLevel(rawConfig.LogLevel)
	if err != nil {
		return config, fmt.Errorf("unknown log level: %s", rawConfig.LogLevel)
	}
	config.LogLevel = logLevel

	config.LogFile = rawConfig.LogFile
	config.OpenTelemetryCollectorURL = rawConfig.OpenTelemetryCollectorURL

	var peers []*peer.AddrInfo
	for _, p := range rawConfig.MpcConfig.Peers {
		addrInfo, err := peer.AddrInfoFromString(p.PeerAddress)
		if err != nil {
			return config, fmt.Errorf("invalid peer address %s: %w", p.PeerAddress, err)
		}
		peers = append(peers, addrInfo)
	}
	config.MpcConfig.Peers = peers
	config.MpcConfig.Port = rawConfig.MpcConfig.Port

	return config, nil
}
