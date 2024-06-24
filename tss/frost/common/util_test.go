// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package common_test

import (
	"testing"

	ecds_common "github.com/ChainSafe/sygma-relayer/tss/ecdsa/common"
	"github.com/ChainSafe/sygma-relayer/tss/frost/common"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/stretchr/testify/suite"
)

type PartyFromPeersTestSuite struct {
	suite.Suite
}

func TestRunPartyFromPeersTestSuite(t *testing.T) {
	suite.Run(t, new(PartyFromPeersTestSuite))
}

func (s *PartyFromPeersTestSuite) Test_SortingParties() {
	// 9tmwQQGnb86t9NKjXUAYkZwNhCnmE478jfagJ1NtBTCFVdiiVUhM2Vb2wSsLAn7
	party1 := ecds_common.CreatePartyID("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")
	// 9tmzMDjd7qftVub5kfPz3wTgckpEMGrg7upHFX47ptoMtyJmyvHQni7iDQWA4Sd
	party2 := ecds_common.CreatePartyID("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")

	peers := peer.IDSlice{peer.ID(party2.Id), peer.ID(party1.Id)}

	prs := common.PartyIDSFromPeers(peers)

	// Assert that the first element in prs matches the string representation of expected peer.ID
	s.Equal("9tmwQQGnb86t9NKjXUAYkZwNhCnmE478jfagJ1NtBTCFVdiiVUhM2Vb2wSsLAn7", string(prs[0]), "First element should match expectedPeer1")

	// Assert that the second element in prs matches the string representation of expected peer.ID
	s.Equal("9tmzMDjd7qftVub5kfPz3wTgckpEMGrg7upHFX47ptoMtyJmyvHQni7iDQWA4Sd", string(prs[1]), "Second element should match expectedPeer2")

}
