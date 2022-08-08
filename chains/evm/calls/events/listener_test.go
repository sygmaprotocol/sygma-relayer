package events_test

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/ChainSafe/sygma/chains/evm/calls/events"
	mock_listener "github.com/ChainSafe/sygma/chains/evm/calls/events/mock"
)

type ListenerTestSuite struct {
	suite.Suite
	mockClient *mock_listener.MockChainClient
	listener   *events.Listener
}

func TestRunListenerTestSuite(t *testing.T) {
	suite.Run(t, new(ListenerTestSuite))
}

func (s *ListenerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.mockClient = mock_listener.NewMockChainClient(ctrl)
	s.listener = events.NewListener(s.mockClient)
}

func (s *ListenerTestSuite) Test_FetchDepositEvent_FetchingTxFails() {
	s.mockClient.EXPECT().WaitAndReturnTxReceipt(common.HexToHash("0xf25ed4a14bf7ad20354b46fe38d7d4525f2ea3042db9a9954ef8d73c558b500c")).Return(nil, fmt.Errorf("error"))

	_, err := s.listener.FetchDepositEvent(events.RetryEvent{TxHash: "0xf25ed4a14bf7ad20354b46fe38d7d4525f2ea3042db9a9954ef8d73c558b500c"}, common.Address{}, big.NewInt(5))

	s.NotNil(err)
}

func (s *ListenerTestSuite) Test_FetchDepositEvent_EventTooNew() {
	s.mockClient.EXPECT().WaitAndReturnTxReceipt(common.HexToHash("0xf25ed4a14bf7ad20354b46fe38d7d4525f2ea3042db9a9954ef8d73c558b500c")).Return(&types.Receipt{
		BlockNumber: big.NewInt(14),
	}, nil)
	s.mockClient.EXPECT().LatestBlock().Return(big.NewInt(10), nil)

	_, err := s.listener.FetchDepositEvent(
		events.RetryEvent{TxHash: "0xf25ed4a14bf7ad20354b46fe38d7d4525f2ea3042db9a9954ef8d73c558b500c"},
		common.HexToAddress("0x5798e01f4b1d8f6a5d91167414f3a915d021bc4a"),
		big.NewInt(5),
	)

	s.NotNil(err)
}

func (s *ListenerTestSuite) Test_FetchDepositEvent_BridgeAndDepositHashMismatch() {
	s.mockClient.EXPECT().WaitAndReturnTxReceipt(common.HexToHash("0xf25ed4a14bf7ad20354b46fe38d7d4525f2ea3042db9a9954ef8d73c558b500c")).Return(&types.Receipt{
		BlockNumber: big.NewInt(11),
	}, nil)
	s.mockClient.EXPECT().LatestBlock().Return(big.NewInt(5), nil)
	tx := types.NewTransaction(1, common.HexToAddress("0x1ec6b294902d42fee964d29fa962e5976e71e67d"), big.NewInt(0), 100, big.NewInt(1), []byte{})
	s.mockClient.EXPECT().TransactionByHash(context.Background(), common.HexToHash("0xf25ed4a14bf7ad20354b46fe38d7d4525f2ea3042db9a9954ef8d73c558b500c")).Return(tx, false, nil)

	_, err := s.listener.FetchDepositEvent(
		events.RetryEvent{TxHash: "0xf25ed4a14bf7ad20354b46fe38d7d4525f2ea3042db9a9954ef8d73c558b500c"},
		common.HexToAddress("0x5798e01f4b1d8f6a5d91167414f3a915d021bc4a"),
		big.NewInt(5),
	)

	s.NotNil(err)
}

func (s *ListenerTestSuite) Test_FetchDepositEvent_ValidEvent() {
	s.mockClient.EXPECT().WaitAndReturnTxReceipt(common.HexToHash("0xf25ed4a14bf7ad20354b46fe38d7d4525f2ea3042db9a9954ef8d73c558b500c")).Return(&types.Receipt{
		BlockNumber: big.NewInt(14),
	}, nil)
	s.mockClient.EXPECT().LatestBlock().Return(big.NewInt(5), nil)
	tx := types.NewTransaction(1, common.HexToAddress("0x5798e01f4b1d8f6a5d91167414f3a915d021bc4a"), big.NewInt(0), 100, big.NewInt(1), []byte{})
	s.mockClient.EXPECT().TransactionByHash(context.Background(), common.HexToHash("0xf25ed4a14bf7ad20354b46fe38d7d4525f2ea3042db9a9954ef8d73c558b500c")).Return(tx, false, nil)

	_, err := s.listener.FetchDepositEvent(
		events.RetryEvent{TxHash: "0xf25ed4a14bf7ad20354b46fe38d7d4525f2ea3042db9a9954ef8d73c558b500c"},
		common.HexToAddress("0x5798e01f4b1d8f6a5d91167414f3a915d021bc4a"),
		big.NewInt(5),
	)

	s.Nil(err)
}
