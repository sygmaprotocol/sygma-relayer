package p2p

import (
	"github.com/ChainSafe/chainbridge-core/communication/p2p/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"testing"
)

type StreamManagerTestSuite struct {
	suite.Suite
	mockController *gomock.Controller
}

func TestRunStreamManagerTestSuite(t *testing.T) {
	suite.Run(t, new(StreamManagerTestSuite))
}

func (s *StreamManagerTestSuite) SetupSuite()    {}
func (s *StreamManagerTestSuite) TearDownSuite() {}
func (s *StreamManagerTestSuite) SetupTest() {
	s.mockController = gomock.NewController(s.T())
}
func (s *StreamManagerTestSuite) TearDownTest() {}

func (s *StreamManagerTestSuite) TestStreamManager_ManagingSubscriptions_Success() {
	streamManager := NewStreamManager()

	stream1 := mock_network.NewMockStream(s.mockController)
	stream2 := mock_network.NewMockStream(s.mockController)
	stream3 := mock_network.NewMockStream(s.mockController)

	streamManager.AddStream("1", stream1)
	streamManager.AddStream("1", stream2)
	streamManager.AddStream("2", stream3)

	s.Len(streamManager.unusedStreams, 2)
	s.Len(streamManager.unusedStreams["1"], 2)
	s.Len(streamManager.unusedStreams["2"], 1)

	stream1.EXPECT().Reset().Times(1).Return(nil)
	stream2.EXPECT().Reset().Times(1).Return(nil)

	streamManager.ReleaseStream("1")

	s.Len(streamManager.unusedStreams, 1)
	s.Len(streamManager.unusedStreams["1"], 0)
	s.Len(streamManager.unusedStreams["2"], 1)
}

func (s *StreamManagerTestSuite) TestStreamManager_ManagingSubscriptionsWithUnknownSession_Success() {
	streamManager := NewStreamManager()

	stream1 := mock_network.NewMockStream(s.mockController)
	stream2 := mock_network.NewMockStream(s.mockController)
	stream3 := mock_network.NewMockStream(s.mockController)

	streamManager.AddStream("1", stream1)
	streamManager.AddStream("2", stream2)
	streamManager.AddStream("UNKNOWN", stream3)

	s.Len(streamManager.unusedStreams, 3)
	s.Len(streamManager.unusedStreams["1"], 1)
	s.Len(streamManager.unusedStreams["2"], 1)

	stream1.EXPECT().Reset().Times(1).Return(nil)
	stream3.EXPECT().Reset().Times(1).Return(nil)

	streamManager.ReleaseStream("1")

	s.Len(streamManager.unusedStreams, 2)
	s.Len(streamManager.unusedStreams["1"], 0)
	s.Len(streamManager.unusedStreams["2"], 1)
	s.Len(streamManager.unusedStreams["UNKNOWN"], 1)
}
