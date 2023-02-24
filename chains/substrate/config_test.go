// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1
package substrate

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/chainbridge-core/config/chain"
	"github.com/stretchr/testify/suite"
)

type NewSubstrateConfigTestSuite struct {
	suite.Suite
}

func TestRunNewSubstrateConfigTestSuite(t *testing.T) {
	suite.Run(t, new(NewSubstrateConfigTestSuite))
}

func (s *NewSubstrateConfigTestSuite) SetupSuite()    {}
func (s *NewSubstrateConfigTestSuite) TearDownSuite() {}
func (s *NewSubstrateConfigTestSuite) SetupTest()     {}
func (s *NewSubstrateConfigTestSuite) TearDownTest()  {}

func (s *NewSubstrateConfigTestSuite) Test_FailedDecode() {
	_, err := NewSubstrateConfig(map[string]interface{}{
		"startBlock": "invalid",
	})

	s.NotNil(err)
}

func (s *NewSubstrateConfigTestSuite) Test_FailedGeneralConfigValidation() {
	_, err := NewSubstrateConfig(map[string]interface{}{})

	s.NotNil(err)
}

func (s *NewSubstrateConfigTestSuite) Test_InvalidBlockConfirmation() {
	_, err := NewSubstrateConfig(map[string]interface{}{
		"id":                 1,
		"endpoint":           "ws://domain.com",
		"name":               "substrate1",
		"blockConfirmations": -1,
	})

	s.NotNil(err)
	s.Equal(err.Error(), "blockConfirmations has to be >=1")
}

func (s *NewSubstrateConfigTestSuite) Test_ValidConfig() {
	rawConfig := map[string]interface{}{
		"id":               1,
		"chainID":          5,
		"substrateNetwork": 0,
		"endpoint":         "ws://domain.com",
		"name":             "substrate1",
	}

	actualConfig, err := NewSubstrateConfig(rawConfig)

	id := new(uint8)
	*id = 1
	s.Nil(err)
	s.Equal(*actualConfig, SubstrateConfig{
		GeneralChainConfig: chain.GeneralChainConfig{
			Name:     "substrate1",
			Endpoint: "ws://domain.com",
			Id:       id,
		},
		StartBlock:         big.NewInt(0),
		ChainID:            big.NewInt(5),
		SubstrateNetwork:   uint8(0),
		BlockConfirmations: big.NewInt(10),
		BlockInterval:      big.NewInt(5),
		BlockRetryInterval: time.Duration(5) * time.Second,
	})
}

func (s *NewSubstrateConfigTestSuite) Test_ValidConfigWithCustomParams() {
	rawConfig := map[string]interface{}{
		"id":                 1,
		"endpoint":           "ws://domain.com",
		"name":               "substrate1",
		"chainID":            5,
		"substrateNetwork":   0,
		"startBlock":         1000,
		"blockConfirmations": 10,
		"blockRetryInterval": 10,
		"blockInterval":      2,
	}

	actualConfig, err := NewSubstrateConfig(rawConfig)

	id := new(uint8)
	*id = 1
	s.Nil(err)
	s.Equal(*actualConfig, SubstrateConfig{
		GeneralChainConfig: chain.GeneralChainConfig{
			Name:     "substrate1",
			Endpoint: "ws://domain.com",
			Id:       id,
		},
		ChainID:            big.NewInt(5),
		SubstrateNetwork:   uint8(0),
		StartBlock:         big.NewInt(1000),
		BlockConfirmations: big.NewInt(10),
		BlockInterval:      big.NewInt(2),
		BlockRetryInterval: time.Duration(10) * time.Second,
	})
}
