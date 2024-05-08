// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package btc

import (
	"encoding/hex"
	"math/big"
	"time"

	"github.com/ChainSafe/chainbridge-core/config/chain"
	"github.com/creasty/defaults"
	"github.com/mitchellh/mapstructure"
)

type RawBtcConfig struct {
	chain.GeneralChainConfig `mapstructure:",squash"`
	ChainID                  int64  `mapstructure:"chainID"`
	StartBlock               int64  `mapstructure:"startBlock"`
	BlockInterval            int64  `mapstructure:"blockInterval" default:"5"`
	BlockRetryInterval       uint64 `mapstructure:"blockRetryInterval" default:"5"`
	BtcNetwork               int64  `mapstructure:"BtcNetwork"`
	Tip                      uint64 `mapstructure:"tip"`
	Address                  string `mapstructure:"address"`
	Tweak                    string `mapstructure:"tweak"`
	Script                   string `mapstructure:"script"`
}

type BtcConfig struct {
	GeneralChainConfig chain.GeneralChainConfig
	ChainID            *big.Int
	StartBlock         *big.Int
	BlockInterval      *big.Int
	Address            string
	Tweak              string
	Script             []byte

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
	scriptBytes, err := hex.DecodeString(c.Script)
	if err != nil {
		return nil, err
	}
	config := &BtcConfig{
		GeneralChainConfig: c.GeneralChainConfig,
		ChainID:            big.NewInt(3),
		BlockRetryInterval: time.Duration(5000) * time.Second,
		StartBlock:         big.NewInt(80000),
		BlockInterval:      big.NewInt(1000000),
		Address:            c.Address,
		Tweak:              c.Tweak,
		Script:             scriptBytes,
	}

	return config, nil
}
