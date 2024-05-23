// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package btc

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ChainSafe/sygma-relayer/config/chain"
	"github.com/creasty/defaults"
	"github.com/mitchellh/mapstructure"
)

type RawBtcConfig struct {
	chain.GeneralChainConfig `mapstructure:",squash"`
	Resources                []ResourceConfig `mapstrcture:"resources"`
	StartBlock               int64            `mapstructure:"startBlock"`
	Username                 string           `mapstructure:"username"`
	Password                 string           `mapstructure:"password"`
	BlockInterval            int64            `mapstructure:"blockInterval" default:"5"`
	BlockRetryInterval       uint64           `mapstructure:"blockRetryInterval" default:"5"`
	BlockConfirmations       int64            `mapstructure:"blockConfirmations" default:"10"`
}

func (c *RawBtcConfig) Validate() error {
	if err := c.GeneralChainConfig.Validate(); err != nil {
		return err
	}

	if c.BlockConfirmations != 0 && c.BlockConfirmations < 1 {
		return fmt.Errorf("blockConfirmations has to be >=1")
	}
	return nil
}

type ResourceConfig struct {
	Address    string
	ResourceID [32]byte
}

type BtcConfig struct {
	GeneralChainConfig chain.GeneralChainConfig
	Bridge             string
	Resources          []ResourceConfig
	StartBlock         *big.Int
	BlockInterval      *big.Int
	Username           string
	Password           string
	BlockRetryInterval time.Duration
	BlockConfirmations *big.Int
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
		Resources:          c.Resources,
		GeneralChainConfig: c.GeneralChainConfig,
		BlockRetryInterval: time.Duration(c.BlockInterval) * time.Second,
		StartBlock:         big.NewInt(c.StartBlock),
		BlockInterval:      big.NewInt(c.BlockInterval),
		BlockConfirmations: big.NewInt(c.BlockConfirmations),
	}

	return config, nil
}
