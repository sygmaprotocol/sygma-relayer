package evm

import (
	"fmt"
	"math/big"
	"time"

	"github.com/creasty/defaults"
	"github.com/mitchellh/mapstructure"

	"github.com/ChainSafe/chainbridge-core/config/chain"
)

type HandlerConfig struct {
	Address string
	Type    string
}

type EVMConfig struct {
	GeneralChainConfig chain.GeneralChainConfig
	Bridge             string
	Handlers           []HandlerConfig
	MaxGasPrice        *big.Int
	GasMultiplier      *big.Float
	GasLimit           *big.Int
	StartBlock         *big.Int
	BlockConfirmations *big.Int
	BlockInterval      *big.Int
	BlockRetryInterval time.Duration
}

type RawEVMConfig struct {
	chain.GeneralChainConfig `mapstructure:",squash"`
	Bridge                   string          `mapstructure:"bridge"`
	Handlers                 []HandlerConfig `mapstrcture:"handlers"`
	MaxGasPrice              int64           `mapstructure:"maxGasPrice" default:"500000000000"`
	GasMultiplier            float64         `mapstructure:"gasMultiplier" default:"1"`
	GasLimit                 int64           `mapstructure:"gasLimit" default:"2000000"`
	StartBlock               int64           `mapstructure:"startBlock"`
	BlockConfirmations       int64           `mapstructure:"blockConfirmations" default:"10"`
	BlockInterval            int64           `mapstructure:"blockInterval" default:"5"`
	BlockRetryInterval       uint64          `mapstructure:"blockRetryInterval" default:"5"`
}

func (c *RawEVMConfig) Validate() error {
	if err := c.GeneralChainConfig.Validate(); err != nil {
		return err
	}
	if c.Bridge == "" {
		return fmt.Errorf("required field chain.Bridge empty for chain %v", *c.Id)
	}
	if c.BlockConfirmations != 0 && c.BlockConfirmations < 1 {
		return fmt.Errorf("blockConfirmations has to be >=1")
	}
	return nil
}

// NewEVMConfig decodes and validates an instance of an EVMConfig from
// raw chain config
func NewEVMConfig(chainConfig map[string]interface{}) (*EVMConfig, error) {
	var c RawEVMConfig
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
	config := &EVMConfig{
		GeneralChainConfig: c.GeneralChainConfig,
		Handlers:           c.Handlers,
		Bridge:             c.Bridge,
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
