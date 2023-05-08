// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/creasty/defaults"
	"github.com/imdario/mergo"

	"github.com/ChainSafe/sygma-relayer/config/relayer"
	"github.com/spf13/viper"
)

type Config struct {
	RelayerConfig relayer.RelayerConfig
	ChainConfigs  []map[string]interface{}
}

type RawConfig struct {
	RelayerConfig relayer.RawRelayerConfig `mapstructure:"relayer" json:"relayer"`
	ChainConfigs  []map[string]interface{} `mapstructure:"domains" json:"domains"`
}

// GetConfigFromENV reads config from Env variables, validates it and parses
// it into config suitable for application
//
// Properties of RelayerConfig are expected to be defined as separate Env variables
// where Env variable name reflects properties position in structure. Each Env variable needs to be prefixed with SYG.
//
// For example, if you want to set Config.RelayerConfig.MpcConfig.Port this would
// translate to Env variable named SYG_RELAYER_MPCCONFIG_PORT.
func GetConfigFromENV(config *Config) (*Config, error) {
	rawConfig, err := loadFromEnv()
	if err != nil {
		return config, err
	}

	return processRawConfig(rawConfig, config)
}

// GetConfigFromFile reads config from file, validates it and parses
// it into config suitable for application
func GetConfigFromFile(path string, config *Config) (*Config, error) {
	rawConfig := RawConfig{}

	viper.SetConfigFile(path)
	viper.SetConfigType("json")

	err := viper.ReadInConfig()
	if err != nil {
		return config, err
	}

	err = viper.Unmarshal(&rawConfig)
	if err != nil {
		return config, err
	}

	return processRawConfig(rawConfig, config)
}

// GetSharedConfigFromNetwork fetches shared configuration from URL and parses it.
func GetSharedConfigFromNetwork(url string, config *Config) (*Config, error) {
	rawConfig := RawConfig{}

	resp, err := http.Get(url)
	if err != nil {
		return &Config{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &Config{}, err
	}

	err = json.Unmarshal(body, &rawConfig)
	if err != nil {
		return &Config{}, err
	}

	config.ChainConfigs = rawConfig.ChainConfigs
	return config, err
}

func processRawConfig(rawConfig RawConfig, config *Config) (*Config, error) {
	if err := defaults.Set(&rawConfig); err != nil {
		return config, err
	}

	relayerConfig, err := relayer.NewRelayerConfig(rawConfig.RelayerConfig)
	if err != nil {
		return config, err
	}

	chainConfigs := make([]map[string]interface{}, 0)
	for i, chain := range rawConfig.ChainConfigs {
		if i < len(config.ChainConfigs) {
			err := mergo.Merge(&chain, config.ChainConfigs[i])
			if err != nil {
				return config, err
			}
		}

		if chain["type"] == "" || chain["type"] == nil {
			return config, fmt.Errorf("chain 'type' must be provided for every configured chain")
		}
		chainConfigs = append(chainConfigs, chain)
	}

	config.ChainConfigs = chainConfigs
	config.RelayerConfig = relayerConfig
	return config, nil
}
