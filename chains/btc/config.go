// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package btc

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/ChainSafe/sygma-relayer/config/chain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/creasty/defaults"
	"github.com/mitchellh/mapstructure"
)

type RawBtcConfig struct {
	chain.GeneralChainConfig `mapstructure:",squash"`
	Resources                []Resource `mapstrcture:"resources"`
	StartBlock               int64      `mapstructure:"startBlock"`
	Username                 string     `mapstructure:"username"`
	Password                 string     `mapstructure:"password"`
	BlockInterval            int64      `mapstructure:"blockInterval" default:"5"`
	BlockRetryInterval       uint64     `mapstructure:"blockRetryInterval" default:"5"`
	BlockConfirmations       int64      `mapstructure:"blockConfirmations" default:"10"`
	Network                  string     `mapstructure:"network" default:"mainnet"`
	Tip                      uint64     `mapstructure:"tip"`
	Tweak                    string     `mapstructure:"tweak"`
	Script                   string     `mapstructure:"script"`
	MempoolUrl               string     `mapstructure:"mempoolUrl"`
}

func (c *RawBtcConfig) Validate() error {
	if err := c.GeneralChainConfig.Validate(); err != nil {
		return err
	}

	if c.BlockConfirmations != 0 && c.BlockConfirmations < 1 {
		return fmt.Errorf("blockConfirmations has to be >=1")
	}

	if c.Username == "" {
		return fmt.Errorf("required field chain.Username empty for chain %v", *c.Id)
	}

	if c.Password == "" {
		return fmt.Errorf("required field chain.Password empty for chain %v", *c.Id)
	}
	return nil
}

type Resource struct {
	Address    string
	ResourceID [32]byte
}

type BtcConfig struct {
	GeneralChainConfig chain.GeneralChainConfig
	Resources          []Resource
	Username           string
	Password           string
	StartBlock         *big.Int
	BlockInterval      *big.Int
	BlockRetryInterval time.Duration
	BlockConfirmations *big.Int
	Tweak              string
	Script             []byte
	MempoolUrl         string
	Network            chaincfg.Params
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

	networkParams, err := networkParams(c.Network)
	if err != nil {
		return nil, err
	}
	config := &BtcConfig{
		GeneralChainConfig: c.GeneralChainConfig,
		StartBlock:         big.NewInt(c.StartBlock),
		BlockConfirmations: big.NewInt(c.BlockConfirmations),
		BlockInterval:      big.NewInt(c.BlockInterval),
		BlockRetryInterval: time.Duration(5000) * time.Second,
		Username:           c.Username,
		Password:           c.Password,
		Network:            networkParams,
		Tweak:              c.Tweak,
		Script:             scriptBytes,
		MempoolUrl:         c.MempoolUrl,
		Resources:          c.Resources,
	}
	return config, nil
}

func networkParams(network string) (chaincfg.Params, error) {
	switch network {
	case "mainnet":
		return chaincfg.MainNetParams, nil
	case "testnet":
		return chaincfg.TestNet3Params, nil
	case "regtest":
		return chaincfg.RegressionNetParams, nil
	default:
		return chaincfg.Params{}, fmt.Errorf("unknown network %s", network)
	}
}
