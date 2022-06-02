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
	PingWaitTime     string `mapstructure:"PingWaitTime" json:"pingWaitTime" default:"1s"`
	PingBackOff      string `mapstructure:"PingBackOff" json:"pingBackOff" default:"1s"`
	PingInterval     string `mapstructure:"PingInterval" json:"pingInterval" default:"1s"`
	ElectionWaitTime string `mapstructure:"ElectionWaitTime" json:"electionWaitTime" default:"2s"`
	BullyWaitTime    string `mapstructure:"BullyWaitTime" json:"bullyWaitTime" default:"25s"`
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

	mpcConfig, err := parseMpcConfig(rawConfig)
	if err != nil {
		return RelayerConfig{}, err
	}
	config.MpcConfig = mpcConfig

	bullyConfig, err := parseBullyConfig(rawConfig)
	if err != nil {
		return RelayerConfig{}, err
	}
	config.BullyConfig = bullyConfig

	return config, nil
}

func parseMpcConfig(rawConfig RawRelayerConfig) (MpcRelayerConfig, error) {
	var mpcConfig MpcRelayerConfig
	var peers []*peer.AddrInfo
	for _, p := range rawConfig.MpcConfig.Peers {
		addrInfo, err := peer.AddrInfoFromString(p.PeerAddress)
		if err != nil {
			return mpcConfig, fmt.Errorf("invalid peer address %s: %w", p.PeerAddress, err)
		}
		peers = append(peers, addrInfo)
	}

	mpcConfig.Peers = peers
	mpcConfig.Port = rawConfig.MpcConfig.Port
	mpcConfig.KeysharePath = rawConfig.MpcConfig.KeysharePath
	mpcConfig.KeystorePath = rawConfig.MpcConfig.KeystorePath
	mpcConfig.Threshold = rawConfig.MpcConfig.Threshold

	return mpcConfig, nil
}

func parseBullyConfig(rawConfig RawRelayerConfig) (BullyConfig, error) {
	electionWaitTime, err := time.ParseDuration(rawConfig.BullyConfig.ElectionWaitTime)
	if err != nil {
		return BullyConfig{}, fmt.Errorf("unable to parse bully election wait time: %w", err)
	}

	pingWaitTime, err := time.ParseDuration(rawConfig.BullyConfig.PingWaitTime)
	if err != nil {
		return BullyConfig{}, fmt.Errorf("unable to parse bully ping wait time: %w", err)
	}

	pingInterval, err := time.ParseDuration(rawConfig.BullyConfig.PingInterval)
	if err != nil {
		return BullyConfig{}, fmt.Errorf("unable to parse bully ping interval time: %w", err)
	}

	pingBackOff, err := time.ParseDuration(rawConfig.BullyConfig.PingBackOff)
	if err != nil {
		return BullyConfig{}, fmt.Errorf("unable to parse bully ping back off time: %w", err)
	}

	bullyWaitTime, err := time.ParseDuration(rawConfig.BullyConfig.BullyWaitTime)
	if err != nil {
		return BullyConfig{}, fmt.Errorf("unable to parse bully wait time: %w", err)
	}

	return BullyConfig{
		PingWaitTime:     pingWaitTime,
		PingBackOff:      pingBackOff,
		PingInterval:     pingInterval,
		ElectionWaitTime: electionWaitTime,
		BullyWaitTime:    bullyWaitTime,
	}, nil
}
