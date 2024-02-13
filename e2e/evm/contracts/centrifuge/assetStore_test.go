package centrifuge_test

import (
	"errors"
	"testing"

	"github.com/ChainSafe/sygma-relayer/e2e/evm/contracts/centrifuge"

	"github.com/sygmaprotocol/sygma-core/mock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type IsCentrifugeAssetStoredTestSuite struct {
	suite.Suite
	gomockController          *gomock.Controller
	mockClient                *mock.MockClient
	mockTransactor            *mock.MockTransactor
	assetStoreContractAddress common.Address
	assetStoreContract        *centrifuge.AssetStoreContract
}

func TestRunIsCentrifugeAssetStoredTestSuite(t *testing.T) {
	suite.Run(t, new(IsCentrifugeAssetStoredTestSuite))
}

func (s *IsCentrifugeAssetStoredTestSuite) SetupSuite()    {}
func (s *IsCentrifugeAssetStoredTestSuite) TearDownSuite() {}
func (s *IsCentrifugeAssetStoredTestSuite) SetupTest() {
	s.gomockController = gomock.NewController(s.T())
	s.mockClient = mock.NewMockClient(s.gomockController)
	s.mockTransactor = mock.NewMockTransactor(s.gomockController)
	s.assetStoreContractAddress = common.HexToAddress("0x9A0E6F91E6031C08326764655432f8F9c180fBa0")
	s.assetStoreContract = centrifuge.NewAssetStoreContract(
		s.mockClient, s.assetStoreContractAddress, s.mockTransactor,
	)
}
func (s *IsCentrifugeAssetStoredTestSuite) TearDownTest() {}

func (s *IsCentrifugeAssetStoredTestSuite) TestCallContractFails() {
	s.mockClient.EXPECT().CallContract(
		gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte{}, errors.New("error"))
	s.mockClient.EXPECT().From().Times(1).Return(common.Address{})
	isStored, err := s.assetStoreContract.IsCentrifugeAssetStored([32]byte{})

	s.NotNil(err)
	s.Equal(isStored, false)
}

func (s *IsCentrifugeAssetStoredTestSuite) TestUnpackingInvalidOutput() {
	s.mockClient.EXPECT().CallContract(
		gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("invalid"), nil)
	s.mockClient.EXPECT().From().Times(1).Return(common.Address{})
	isStored, err := s.assetStoreContract.IsCentrifugeAssetStored([32]byte{})

	s.NotNil(err)
	s.Equal(isStored, false)
}

func (s *IsCentrifugeAssetStoredTestSuite) TestEmptyOutput() {
	s.mockClient.EXPECT().CallContract(
		gomock.Any(), gomock.Any(), gomock.Any(),
	).Return([]byte{}, nil)
	s.mockClient.EXPECT().CodeAt(
		gomock.Any(), gomock.Any(), gomock.Any(),
	).Return(nil, errors.New("error"))
	s.mockClient.EXPECT().From().Times(1).Return(common.Address{})

	isStored, err := s.assetStoreContract.IsCentrifugeAssetStored([32]byte{})

	s.NotNil(err)
	s.Equal(isStored, false)
}

func (s *IsCentrifugeAssetStoredTestSuite) TestValidStoredAsset() {
	response := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	s.mockClient.EXPECT().CallContract(
		gomock.Any(), gomock.Any(), gomock.Any()).Return(response, nil)
	s.mockClient.EXPECT().From().Times(1).Return(common.Address{})

	isStored, err := s.assetStoreContract.IsCentrifugeAssetStored([32]byte{})

	s.Nil(err)
	s.Equal(isStored, true)
}
