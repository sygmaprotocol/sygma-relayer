package evmclient_test

import (
	"testing"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type EvmClientTestSuite struct {
	suite.Suite
	client           *evmclient.EVMClient
	gomockController *gomock.Controller
}

func TestRunEvmClientTestSuite(t *testing.T) {
	suite.Run(t, new(EvmClientTestSuite))
}

func (s *EvmClientTestSuite) SetupSuite() {
	s.gomockController = gomock.NewController(s.T())
	s.client = &evmclient.EVMClient{}
}
func (s *EvmClientTestSuite) TearDownSuite() {}
func (s *EvmClientTestSuite) SetupTest() {
}
func (s *EvmClientTestSuite) TearDownTest() {}

/*

func (s *EvmClientTestSuite) TestUnpackDepositEventLogFailedUnpack() {
	abi, _ := abi.JSON(strings.NewReader(consts.BridgeABI))

	_, err := s.client.UnpackDepositEventLog(abi, []byte("invalid"))

	s.NotNil(err)
}

func (s *EvmClientTestSuite) TestUnpackDepositEventLogValidData() {
	abi, _ := abi.JSON(strings.NewReader(consts.BridgeABI))
	logDataBytes, _ := hex.DecodeString(logData)
	expectedRID := types.ResourceID(types.ResourceID{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xd6, 0x6, 0xa0, 0xc, 0x1a, 0x39, 0xda, 0x53, 0xea, 0x7b, 0xb3, 0xab, 0x57, 0xb, 0xbe, 0x40, 0xb1, 0x56, 0xeb, 0x66, 0x0})

	dl, err := s.client.UnpackDepositEventLog(abi, logDataBytes)

	s.Nil(err)
	s.Equal(dl.SenderAddress.String(), "0x0000000000000000000000000000000000000000")
	s.Equal(dl.DepositNonce, uint64(1))
	s.Equal(dl.DestinationDomainID, uint8(2))
	s.Equal(dl.ResourceID, expectedRID)
	s.Equal(dl.HandlerResponse, []byte{})
}

*/
