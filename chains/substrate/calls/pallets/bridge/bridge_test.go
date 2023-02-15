// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package bridge_test

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/calls/pallets/bridge"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/executor/proposal"
	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/suite"
)

type BridgeTestSuite struct {
	suite.Suite
}

func TestRunPermissionlessGenericHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(BridgeTestSuite))
}

func (s *BridgeTestSuite) Test_FetchDepositEvent_ValidEvent() {
	prop := proposal.Proposal{
		Source:       1,
		DepositNonce: 1,
		ResourceId:   [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Data:         []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 37, 0, 1, 1, 0, 128, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125},
	}
	fmt.Println("byteeeeeees")
	bt := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 37, 0, 1, 1, 0, 128, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125}
	fmt.Println(common.Bytes2Hex(bt))
	pallet := bridge.NewBridgePallet(nil)
	res, err := pallet.ProposalsHash([]*proposal.Proposal{&prop})
	fmt.Println(res)
	s.Nil(err)
	//s.Equal(crypto.Keccak256(rawData), res)
}
