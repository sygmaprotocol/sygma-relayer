// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package btc_test

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/sygma-relayer/chains/btc"
	"github.com/ChainSafe/sygma-relayer/chains/btc/listener"
	"github.com/ChainSafe/sygma-relayer/config/chain"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/common"
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

func (s *NewBtcConfigTestSuite) Test_InvalidUsername() {
	_, err := btc.NewBtcConfig(map[string]interface{}{
		"id":       1,
		"endpoint": "ws://domain.com",
		"name":     "btc1",
		"password": "pass123",

		"blockConfirmations": 1,
	})

	s.NotNil(err)
	s.Equal(err.Error(), "required field chain.Username empty for chain 1")
}

func (s *NewBtcConfigTestSuite) Test_InvalidPassword() {
	_, err := btc.NewBtcConfig(map[string]interface{}{
		"id":       1,
		"endpoint": "ws://domain.com",
		"name":     "btc1",
		"username": "pass123",

		"blockConfirmations": 1,
	})

	s.NotNil(err)
	s.Equal(err.Error(), "required field chain.Password empty for chain 1")
}

func (s *NewBtcConfigTestSuite) Test_ValidConfig() {
	expectedResource := listener.SliceTo32Bytes(common.LeftPadBytes([]byte{3}, 31))
	expectedAddress, _ := btcutil.DecodeAddress("tb1qln69zuhdunc9stwfh6t7adexxrcr04ppy6thgm", &chaincfg.TestNet3Params)
	expectedScript, _ := hex.DecodeString("51206a698882348433b57d549d6344f74500fcd13ad8d2200cdf89f8e39e5cafa7d5")

	rawConfig := map[string]interface{}{
		"id":       1,
		"endpoint": "ws://domain.com",
		"name":     "btc1",
		"username": "username",
		"password": "pass123",
		"network":  "testnet",
		"resources": []interface{}{
			btc.RawResource{
				Address:    "tb1qln69zuhdunc9stwfh6t7adexxrcr04ppy6thgm",
				ResourceID: "0x0000000000000000000000000000000000000000000000000000000000000300",
			},
		},
		"script": "51206a698882348433b57d549d6344f74500fcd13ad8d2200cdf89f8e39e5cafa7d5",
		"tweak":  "tweak",
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
		Username:           "username",
		Password:           "pass123",
		StartBlock:         big.NewInt(0),
		BlockConfirmations: big.NewInt(10),
		BlockInterval:      big.NewInt(5),
		BlockRetryInterval: time.Duration(5) * time.Second,
		Network:            chaincfg.TestNet3Params,
		Resources: []btc.Resource{
			{
				Address:    expectedAddress,
				ResourceID: expectedResource,
			},
		},
		Script: expectedScript,
		Tweak:  "tweak",
	})
}
