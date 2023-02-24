package chains

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"
)

const bridgeVersion = "3.1.0"
const verifyingContract = "6CdE2Cd82a4F8B74693Ff5e194c19CA08c2d1c68"

type UtilTestSuite struct {
	suite.Suite
}

func TestRunNewEVMConfigTestSuite(t *testing.T) {
	suite.Run(t, new(UtilTestSuite))
}

func (s *UtilTestSuite) Test_CalculateStartingBlock_ProperAdjustment() {
	res := CalculateStartingBlock(big.NewInt(104), big.NewInt(5))
	s.Equal(big.NewInt(100), res)
}

func (s *UtilTestSuite) Test_CalculateStartingBlock_NoAdjustment() {
	res := CalculateStartingBlock(big.NewInt(200), big.NewInt(5))
	s.Equal(big.NewInt(200), res)
}

func (s *UtilTestSuite) Test_NewProposalsHash() {
	data := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 90, 243, 16, 122, 64, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 36, 0, 1, 1, 0, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125}
	prop := []*Proposal{NewProposal(1, 2, 3, [32]byte{3}, data)}
	correctRes := []byte{253, 216, 81, 25, 46, 239, 181, 138, 51, 225, 165, 111, 156, 95, 27, 239, 160, 87, 89, 84, 50, 22, 97, 185, 132, 200, 201, 210, 204, 99, 94, 131}

	res, err := NewProposalsHash(prop, 5, verifyingContract, bridgeVersion)
	s.Nil(err)
	s.Equal(correctRes, res)
}
