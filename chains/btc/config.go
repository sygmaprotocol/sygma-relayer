// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package btc

import (
	"math/big"
	"time"

	"github.com/ChainSafe/sygma-relayer/config/chain"
	"github.com/creasty/defaults"
	"github.com/mitchellh/mapstructure"
)

type RawBtcConfig struct {
	chain.GeneralChainConfig `mapstructure:",squash"`
	Bridge                   string `mapstructure:"bridge"`
	StartBlock               int64  `mapstructure:"startBlock"`
	BlockInterval            int64  `mapstructure:"blockInterval" default:"5"`
	BlockRetryInterval       uint64 `mapstructure:"blockRetryInterval" default:"5"`
}

type BtcConfig struct {
	GeneralChainConfig chain.GeneralChainConfig
	Bridge             string
	StartBlock         *big.Int
	BlockInterval      *big.Int
	BlockRetryInterval time.Duration
}

// NewBtcConfig decodes and validates an instance of an BtcConfig from
// raw chain config
func NewBtcConfig(chainConfig map[string]interface{}) (*BtcConfig, error) {
	var c RawBtcConfig
	err := mapstructure.Decode(chainConfig, &c)
	if err != nil {
		return nil, err
	}

	err = defaults.Set(&c)
	if err != nil {
		return nil, err
	}

	err = c.Validate()
	if err != nil {
		return nil, err
	}

	c.GeneralChainConfig.ParseFlags()
	config := &BtcConfig{
		Bridge:             c.Bridge,
		GeneralChainConfig: c.GeneralChainConfig,
		BlockRetryInterval: time.Duration(c.BlockInterval) * time.Second,
		StartBlock:         big.NewInt(c.StartBlock),
		BlockInterval:      big.NewInt(c.BlockInterval),
	}

	return config, nil
}
