// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener_test

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ChainSafe/sygma-relayer/chains/btc/config"
	"github.com/ChainSafe/sygma-relayer/chains/btc/listener"
	mock_listener "github.com/ChainSafe/sygma-relayer/chains/btc/listener/mock"
	"github.com/ChainSafe/sygma-relayer/relayer/transfer"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/sygmaprotocol/sygma-core/relayer/message"

	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type DepositHandlerTestSuite struct {
	suite.Suite
	fungibleTransferEventHandler *listener.FungibleTransferEventHandler
	mockDepositHandler           *mock_listener.MockDepositHandler
	domainID                     uint8
	resource                     config.Resource
	msgChan                      chan []*message.Message
	mockConn                     *mock_listener.MockConnection
	feeAddress                   btcutil.Address
}

func TestRunDepositHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(DepositHandlerTestSuite))
}

func (s *DepositHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.domainID = 1
	address, _ := btcutil.DecodeAddress("tb1pdf5c3q35ssem2l25n435fa69qr7dzwkc6gsqehuflr3euh905l2slafjvv", &chaincfg.TestNet3Params)
	s.feeAddress, _ = btcutil.DecodeAddress("tb1qln69zuhdunc9stwfh6t7adexxrcr04ppy6thgm", &chaincfg.TestNet3Params)

	s.resource = config.Resource{Address: address, ResourceID: [32]byte{}}
	s.mockDepositHandler = mock_listener.NewMockDepositHandler(ctrl)
	s.msgChan = make(chan []*message.Message, 2)
	s.mockConn = mock_listener.NewMockConnection(ctrl)
	s.fungibleTransferEventHandler = listener.NewFungibleTransferEventHandler(zerolog.Context{}, s.domainID, s.mockDepositHandler, s.msgChan, s.mockConn, s.resource, s.feeAddress)
}

func (s *DepositHandlerTestSuite) Test_FetchDepositFails_GetBlockHashError() {
	s.mockConn.EXPECT().GetBlockHash(int64(100)).Return(nil, fmt.Errorf("error"))
	err := s.fungibleTransferEventHandler.HandleEvents(big.NewInt(100))

	s.NotNil(err)
}

func (s *DepositHandlerTestSuite) Test_FetchDepositFails_GetBlockVerboseTxError() {
	hash, _ := chainhash.NewHashFromStr("00000000000000000008bba5a6ff31fdb9bb1d4147905b5b3c47a07a07235bfc")
	s.mockConn.EXPECT().GetBlockHash(int64(100)).Return(hash, nil)
	s.mockConn.EXPECT().GetBlockVerboseTx(hash).Return(nil, fmt.Errorf("error"))

	err := s.fungibleTransferEventHandler.HandleEvents(big.NewInt(100))
	s.NotNil(err)
}

func (s *DepositHandlerTestSuite) Test_CalculateNonceFail_BlockNumberOverflow() {

	blockNumber := new(big.Int)
	blockNumber.SetString("18446744073709551616", 10)
	nonce, err := s.fungibleTransferEventHandler.CalculateNonce(blockNumber, 5)
	s.Equal(nonce, uint64(0))
	s.NotNil(err)
}

func (s *DepositHandlerTestSuite) Test_CalculateNonce() {
	blockNumber := big.NewInt(123)
	nonce, err := s.fungibleTransferEventHandler.CalculateNonce(blockNumber, 4)
	s.Equal(nonce, uint64(1234))
	s.Nil(err)
}

func (s *DepositHandlerTestSuite) Test_HandleDepositFails_ExecutionContinue() {
	blockNumber := big.NewInt(100)
	data2 := map[string]any{
		"deposit_nonce": uint64(1001),
		"resource_id":   [32]byte{0},
		"amount":        big.NewInt(19000),
		"deposit_data":  "0xe9f23A8289764280697a03aC06795eA92a170e42_1",
	}

	dat := strings.Split("0xe9f23A8289764280697a03aC06795eA92a170e42_1", "_")
	evmAdd := common.HexToAddress(dat[0]).Bytes()
	s.mockDepositHandler.EXPECT().HandleDeposit(
		s.domainID,
		data2["deposit_nonce"],
		data2["resource_id"],
		data2["amount"],
		data2["deposit_data"],
		blockNumber,
	).Return(&message.Message{
		Source:      s.domainID,
		Destination: uint8(1),
		Data: []interface{}{
			big.NewInt(19000).Bytes(),
			evmAdd,
		},
		Type: transfer.TransferMessageType,
		ID:   "messageid",
	}, nil)

	d2 := btcjson.TxRawResult{
		Vin: []btcjson.Vin{

			{
				Txid: "00000000000000000008bba5a6ff31fdb9bb1d4147905b5b3c47a07a07235bfc",
			},
		},
		Vout: []btcjson.Vout{
			{
				ScriptPubKey: btcjson.ScriptPubKeyResult{
					Type: "nulldata",
					Hex:  "6a2c3078653966323341383238393736343238303639376130336143303637393565413932613137306534325f31",
				},
			},
			{
				ScriptPubKey: btcjson.ScriptPubKeyResult{
					Type:    "witness_v1_taproot",
					Address: "tb1qln69zuhdunc9stwfh6t7adexxrcr04ppy6thgm",
				},
				Value: float64(0.00019),
			},
		},
	}

	d1 := btcjson.TxRawResult{
		Vin: []btcjson.Vin{

			{
				Txid: "00000000000000000008bba5a6ff31fdb9bb1d4147905b5b3c47a07a07235bfc",
			},
		},
		Vout: []btcjson.Vout{
			{
				ScriptPubKey: btcjson.ScriptPubKeyResult{
					Type: "nulldata",
					Hex:  "6a2c3078653966323341383238393736343238303639376130336143303637393565413932613137306534325f31",
				},
			},
			{
				ScriptPubKey: btcjson.ScriptPubKeyResult{
					Type:    "witness_v1_taproot",
					Address: "invalidBridgeAddressuhdunc9stwfh6t7adexxrcr04ppy6thgm",
				},
				Value: float64(0.00019),
			},
		},
	}

	evts := []btcjson.TxRawResult{d1, d2}
	hash, _ := chainhash.NewHashFromStr("00000000000000000008bba5a6ff31fdb9bb1d4147905b5b3c47a07a07235bfc")
	sampleResult := &btcjson.GetBlockVerboseTxResult{
		Hash:   "00000000000000000008bba5a6ff31fdb9bb1d4147905b5b3c47a07a07235bfc",
		Height: 100,
		Tx:     evts,
	}

	s.mockConn.EXPECT().GetBlockHash(int64(100)).Return(hash, nil)
	s.mockConn.EXPECT().GetBlockVerboseTx(hash).Return(sampleResult, nil)

	err := s.fungibleTransferEventHandler.HandleEvents(blockNumber)
	msgs := <-s.msgChan

	s.Nil(err)
	s.Equal(msgs, []*message.Message{{Source: s.domainID,
		Destination: 1,
		Data: []interface{}{
			big.NewInt(19000).Bytes(),
			evmAdd,
		},
		Type: transfer.TransferMessageType,
		ID:   "messageid"}})
}
