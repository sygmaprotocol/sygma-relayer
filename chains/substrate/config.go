// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package substrate

import (
	"fmt"
	"math/big"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/creasty/defaults"
	"github.com/mitchellh/mapstructure"

	"github.com/ChainSafe/chainbridge-core/config/chain"
)

type RawSubstrateConfig struct {
	chain.GeneralChainConfig `mapstructure:",squash"`
	ChainID                  int64  `mapstructure:"chainID"`
	StartBlock               int64  `mapstructure:"startBlock"`
	BlockConfirmations       int64  `mapstructure:"blockConfirmations" default:"10"`
	BlockInterval            int64  `mapstructure:"blockInterval" default:"5"`
	BlockRetryInterval       uint64 `mapstructure:"blockRetryInterval" default:"5"`
	SubstrateNetwork         int64  `mapstructure:"substrateNetwork"`
	Tip                      uint64 `mapstructure:"tip"`
}

type SubstrateConfig struct {
	GeneralChainConfig chain.GeneralChainConfig
	ChainID            *big.Int
	StartBlock         *big.Int
	BlockConfirmations *big.Int
	BlockInterval      *big.Int
	BlockRetryInterval time.Duration
	SubstrateNetwork   uint8
	Tip                uint64
}

func (c *SubstrateConfig) String() string {
	kp, _ := signature.KeyringPairFromSecret(c.GeneralChainConfig.Key, c.SubstrateNetwork)
	return fmt.Sprintf(`Name: '%s', Id: '%d', Type: '%s', BlockstorePath: '%s', FreshStart: '%t', LatestBlock: '%t', Key address: '%s', StartBlock: '%s', BlockConfirmations: '%s', BlockInterval: '%s', BlockRetryInterval: '%s', ChainID: '%d', Tip: '%d'`,
		c.GeneralChainConfig.Name,
		*c.GeneralChainConfig.Id,
		c.GeneralChainConfig.Type,
		c.GeneralChainConfig.BlockstorePath,
		c.GeneralChainConfig.FreshStart,
		c.GeneralChainConfig.LatestBlock,
		kp.Address,
		c.StartBlock,
		c.BlockConfirmations,
		c.BlockInterval,
		c.BlockRetryInterval,
		c.ChainID,
		c.Tip,
	)
}

func (c *RawSubstrateConfig) Validate() error {
	if err := c.GeneralChainConfig.Validate(); err != nil {
		return err
	}

	if c.BlockConfirmations != 0 && c.BlockConfirmations < 1 {
		return fmt.Errorf("blockConfirmations has to be >=1")
	}
	return nil
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
		ChainID:            big.NewInt(c.ChainID),
		BlockRetryInterval: time.Duration(c.BlockRetryInterval) * time.Second,
		StartBlock:         big.NewInt(c.StartBlock),
		BlockConfirmations: big.NewInt(c.BlockConfirmations),
		BlockInterval:      big.NewInt(c.BlockInterval),
		SubstrateNetwork:   uint8(c.SubstrateNetwork),
		Tip:                uint64(c.Tip),
	}

	return config, nil
}
