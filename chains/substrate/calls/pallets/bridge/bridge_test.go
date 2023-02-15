// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package bridge_test

import (
	"testing"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/calls/pallets/bridge"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/executor/proposal"

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
		Data:         []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 20, 92, 31, 89, 97, 105, 107, 173, 46, 115, 247, 52, 23, 240, 126, 245, 92, 98, 162, 220, 91},
	}

	pallet := bridge.NewBridgePallet(nil)
	pallet.ProposalsHash([]*proposal.Proposal{&prop})
}
