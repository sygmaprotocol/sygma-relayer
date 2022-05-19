package relayer

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog"
	"time"
)

type RelayerConfig struct {
	OpenTelemetryCollectorURL string
	LogLevel                  zerolog.Level
	LogFile                   string
	MpcConfig                 MpcRelayerConfig
	BullyConfig               BullyConfig
}

type MpcRelayerConfig struct {
	Peers        []*peer.AddrInfo
	Port         uint16
	KeysharePath string
	KeystorePath string
	Threshold    int
}

type BullyConfig struct {
	PingWaitTime     time.Duration
	PingBackOff      time.Duration
	PingInterval     time.Duration
	ElectionWaitTime time.Duration
	BullyWaitTime    time.Duration
}

type RawRelayerConfig struct {
	OpenTelemetryCollectorURL string              `mapstructure:"OpenTelemetryCollectorURL" json:"opentelemetryCollectorURL"`
	LogLevel                  string              `mapstructure:"LogLevel" json:"logLevel" default:"info"`
	LogFile                   string              `mapstructure:"LogFile" json:"logFile" default:"out.log"`
	MpcConfig                 RawMpcRelayerConfig `mapstructure:"MpcConfig" json:"mpcConfig"`
	BullyConfig               RawBullyConfig      `mapstructure:"BullyConfig" json:"bullyConfig"`
}

type RawMpcRelayerConfig struct {
	KeysharePath string    `mapstructure:"KeysharePath" json:"keysharePath"`
	KeystorePath string    `mapstructure:"KeystorePath" json:"keystorePath"`
	Threshold    int       `mapstructure:"Threshold" json:"threshold"`
	Peers        []RawPeer `mapstructure:"Peers" json:"peers"`
	Port         uint16    `mapstructure:"Port" json:"port" default:"9000"`
}

type RawPeer struct {
	PeerAddress string `mapstructure:"PeerAddress" json:"peerAddress"`
}

type RawBullyConfig struct {
	PingWaitTime     string `mapstructure:"PingWaitTime" json:"pingWaitTime"`
	PingBackOff      string `mapstructure:"PingBackOff" json:"pingBackOff"`
	PingInterval     string `mapstructure:"PingInterval" json:"pingInterval"`
	ElectionWaitTime string `mapstructure:"ElectionWaitTime" json:"electionWaitTime"`
	BullyWaitTime    string `mapstructure:"BullyWaitTime" json:"bullyWaitTime"`
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
	config.MpcConfig.KeysharePath = rawConfig.MpcConfig.KeysharePath
	config.MpcConfig.KeystorePath = rawConfig.MpcConfig.KeystorePath
	config.MpcConfig.Threshold = rawConfig.MpcConfig.Threshold

	electionWaitTime, err := time.ParseDuration(rawConfig.BullyConfig.ElectionWaitTime)
	if err != nil {
		return RelayerConfig{}, fmt.Errorf("unable to parse bully election wait time: %w", err)
	}

	pingWaitTime, err := time.ParseDuration(rawConfig.BullyConfig.PingWaitTime)
	if err != nil {
		return RelayerConfig{}, fmt.Errorf("unable to parse bully ping wait time: %w", err)
	}

	pingInterval, err := time.ParseDuration(rawConfig.BullyConfig.PingInterval)
	if err != nil {
		return RelayerConfig{}, fmt.Errorf("unable to parse bully ping interval time: %w", err)
	}

	pingBackOff, err := time.ParseDuration(rawConfig.BullyConfig.PingBackOff)
	if err != nil {
		return RelayerConfig{}, fmt.Errorf("unable to parse bully election wait time: %w", err)
	}

	config.BullyConfig = BullyConfig{
		PingWaitTime:     pingWaitTime,
		PingBackOff:      pingBackOff,
		PingInterval:     pingInterval,
		ElectionWaitTime: electionWaitTime,
	}

	return config, nil
}
