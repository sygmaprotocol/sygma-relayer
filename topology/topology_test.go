package topology

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type StorageTestSuite struct {
	suite.Suite
}

func TestRunStorageTestSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}

func (s *StorageTestSuite) SetupSuite()    {}
func (s *StorageTestSuite) TearDownSuite() {}
func (s *StorageTestSuite) SetupTest()     {}
func (s *StorageTestSuite) TearDownTest()  {}

func (s *StorageTestSuite) Test() {

}
