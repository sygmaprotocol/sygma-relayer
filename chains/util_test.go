package chains

import (
	"github.com/stretchr/testify/suite"
	"math/big"
	"testing"
)

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
