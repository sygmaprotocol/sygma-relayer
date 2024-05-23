// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package btc_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/sygma-relayer/chains/btc"
	"github.com/ChainSafe/sygma-relayer/config/chain"
	"github.com/stretchr/testify/suite"
)

type NewBtcConfigTestSuite struct {
	suite.Suite
}

func TestRunNewBtcConfigTestSuite(t *testing.T) {
	suite.Run(t, new(NewBtcConfigTestSuite))
}

func (s *NewBtcConfigTestSuite) Test_FailedDecode() {
	_, err := btc.NewBtcConfig(map[string]interface{}{
		"gasLimit": "invalid",
	})

	s.NotNil(err)
}

func (s *NewBtcConfigTestSuite) Test_FailedGeneralConfigValidation() {
	_, err := btc.NewBtcConfig(map[string]interface{}{})

	s.NotNil(err)
}

func (s *NewBtcConfigTestSuite) Test_FailedBtcConfigValidation() {
	_, err := btc.NewBtcConfig(map[string]interface{}{
		"id":       1,
		"endpoint": "",
		"name":     "btc1",
	})

	s.NotNil(err)
}

func (s *NewBtcConfigTestSuite) Test_InvalidBlockConfirmation() {
	_, err := btc.NewBtcConfig(map[string]interface{}{
		"id":                 1,
		"endpoint":           "ws://domain.com",
		"name":               "btc1",
		"blockConfirmations": -1,
	})

	s.NotNil(err)
	s.Equal(err.Error(), "blockConfirmations has to be >=1")
}

func (s *NewBtcConfigTestSuite) Test_ValidConfig() {
	rawConfig := map[string]interface{}{
		"id":       1,
		"endpoint": "ws://domain.com",
		"name":     "btc1",
	}

	actualConfig, err := btc.NewBtcConfig(rawConfig)

	id := new(uint8)
	*id = 1
	s.Nil(err)
	s.Equal(*actualConfig, btc.BtcConfig{
		GeneralChainConfig: chain.GeneralChainConfig{
			Name:     "btc1",
			Endpoint: "ws://domain.com",
			Id:       id,
		},
		StartBlock:         big.NewInt(0),
		BlockConfirmations: big.NewInt(10),
		BlockInterval:      big.NewInt(5),
		BlockRetryInterval: time.Duration(5) * time.Second,
	})
}
