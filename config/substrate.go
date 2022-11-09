// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"math/big"
	"time"

	"github.com/creasty/defaults"
	"github.com/mitchellh/mapstructure"

	"github.com/ChainSafe/chainbridge-core/config/chain"
)

type SubstrateConfig struct {
	GeneralChainConfig chain.GeneralChainConfig
	MaxGasPrice        *big.Int
	GasMultiplier      *big.Float
	GasLimit           *big.Int
	StartBlock         *big.Int
	BlockConfirmations *big.Int
	BlockInterval      *big.Int
	BlockRetryInterval time.Duration
}

// NewSubstrateConfig decodes and validates an instance of an SubstrateConfig from
// raw chain config
func NewSubstrateConfig(chainConfig map[string]interface{}) (*SubstrateConfig, error) {
	var c RawSubstrateConfig
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
	config := &SubstrateConfig{
		GeneralChainConfig: c.GeneralChainConfig,
		BlockRetryInterval: time.Duration(c.BlockRetryInterval) * time.Second,
		GasLimit:           big.NewInt(c.GasLimit),
		MaxGasPrice:        big.NewInt(c.MaxGasPrice),
		GasMultiplier:      big.NewFloat(c.GasMultiplier),
		StartBlock:         big.NewInt(c.StartBlock),
		BlockConfirmations: big.NewInt(c.BlockConfirmations),
		BlockInterval:      big.NewInt(c.BlockInterval),
	}

	return config, nil
}

type RawSubstrateConfig struct {
	chain.GeneralChainConfig `mapstructure:",squash"`
	MaxGasPrice              int64   `mapstructure:"maxGasPrice" default:"20000000000"`
	GasMultiplier            float64 `mapstructure:"gasMultiplier" default:"1"`
	GasLimit                 int64   `mapstructure:"gasLimit" default:"2000000"`
	StartBlock               int64   `mapstructure:"startBlock"`
	BlockConfirmations       int64   `mapstructure:"blockConfirmations" default:"10"`
	BlockInterval            int64   `mapstructure:"blockInterval" default:"5"`
	BlockRetryInterval       uint64  `mapstructure:"blockRetryInterval" default:"5"`
}
