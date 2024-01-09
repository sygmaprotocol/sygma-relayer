// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package chains

import (
	"github.com/stretchr/testify/suite"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

const bridgeVersion = "3.1.0"
const verifyingContract = "6CdE2Cd82a4F8B74693Ff5e194c19CA08c2d1c68"

type ProposalTestSuite struct {
	suite.Suite
}

func (s *ProposalTestSuite) Test_ProposalsHash() {
	data := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 90, 243, 16, 122, 64, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 36, 0, 1, 1, 0, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125}
	prop := []*proposal.Proposal{NewTransferProposal(1, 2, 3, [32]byte{3}, nil, data, TransferProposalType)}
	correctRes := []byte{253, 216, 81, 25, 46, 239, 181, 138, 51, 225, 165, 111, 156, 95, 27, 239, 160, 87, 89, 84, 50, 22, 97, 185, 132, 200, 201, 210, 204, 99, 94, 131}

	res, err := ProposalsHash(prop, 5, verifyingContract, bridgeVersion)
	s.Nil(err)
	s.Equal(correctRes, res)
}
