package config

import (
	"fmt"

	"github.com/creasty/defaults"

	"github.com/ChainSafe/sygma-relayer/config/relayer"
	"github.com/spf13/viper"
)

type Config struct {
	RelayerConfig relayer.RelayerConfig
	ChainConfigs  []map[string]interface{}
}

type RawConfig struct {
	RelayerConfig relayer.RawRelayerConfig `mapstructure:"relayer" json:"relayer"`
	ChainConfigs  []map[string]interface{} `mapstructure:"chains" json:"chains"`
}

// GetConfigFromENV reads config from ENV variables, validates it and parses
// it into config suitable for application
//
//
// Properties of RelayerConfig are expected to be defined as separate ENV variables
// where ENV variable name reflects properties position in structure. Each ENV variable needs to be prefixed with CBH.
//
// For example, if you want to set Config.RelayerConfig.MpcConfig.Port this would
// translate to ENV variable named CBH_RELAYER_MPCCONFIG_PORT.
//
//
// Each ChainConfig is defined as one ENV variable, where its content is JSON configuration for one chain/domain.
// Variables are named like this: CBH_DOM_X where X is domain id.
//
func GetConfigFromENV() (Config, error) {
	rawConfig, err := loadFromEnv()
	if err != nil {
		return Config{}, err
	}

	return processRawConfig(rawConfig)
}

// GetConfigFromFile reads config from file, validates it and parses
// it into config suitable for application
func GetConfigFromFile(path string) (Config, error) {
	rawConfig := RawConfig{}

	viper.SetConfigFile(path)
	viper.SetConfigType("json")

	err := viper.ReadInConfig()
	if err != nil {
		return Config{}, err
	}

	err = viper.Unmarshal(&rawConfig)
	if err != nil {
		return Config{}, err
	}

	return processRawConfig(rawConfig)
}

func processRawConfig(rawConfig RawConfig) (Config, error) {
	config := Config{}

	if err := defaults.Set(&rawConfig); err != nil {
		return Config{}, err
	}

	relayerConfig, err := relayer.NewRelayerConfig(rawConfig.RelayerConfig)
	if err != nil {
		return config, err
	}

	for _, chain := range rawConfig.ChainConfigs {
		if chain["type"] == "" || chain["type"] == nil {
			return config, fmt.Errorf("chain 'type' must be provided for every configured chain")
		}
	}

	config.RelayerConfig = relayerConfig
	config.ChainConfigs = rawConfig.ChainConfigs

	return config, nil
}
