package listener_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ChainSafe/sygma-relayer/chains/btc"
	"github.com/ChainSafe/sygma-relayer/chains/btc/listener"
	mock_listener "github.com/ChainSafe/sygma-relayer/chains/btc/listener/mock"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type DecodeEventsSuite struct {
	suite.Suite
	mockConn *mock_listener.MockConnection
	resource btc.Resource
}

func TestRunDecodeDepositEventsSuite(t *testing.T) {
	suite.Run(t, new(DecodeEventsSuite))
}

func (s *DecodeEventsSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.resource = btc.Resource{Address: "tb1qln69zuhdunc9stwfh6t7adexxrcr04ppy6thgm", ResourceID: [32]byte{}}
	s.mockConn = mock_listener.NewMockConnection(ctrl)
}

func (s *DecodeEventsSuite) Test_DecodeDepositEvent_ErrorDecodingOPRETURNData() {
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
					Hex:  "InvalidCharć",
				},
			},
			{
				ScriptPubKey: btcjson.ScriptPubKeyResult{
					Type:    "witness_v0_keyhash",
					Address: "tb1qln69zuhdunc9stwfh6t7adexxrcr04ppy6thgm",
				},
				Value: float64(0.00019),
			},
		},
	}

	deposit, isDeposit, err := listener.DecodeDepositEvent(d1, s.mockConn, s.resource)
	s.Equal(isDeposit, true)
	s.NotNil(err)
	s.Equal(deposit, listener.Deposit{})
}

func (s *DecodeEventsSuite) Test_DecodeDepositEvent_ErrorInvalidPrevTxHash() {

	d1 := btcjson.TxRawResult{
		Vin: []btcjson.Vin{

			{
				Txid: "invalidTXIDć",
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
					Type:    "witness_v0_keyhash",
					Address: "tb1qln69zuhdunc9stwfh6t7adexxrcr04ppy6thgm",
				},
				Value: float64(0.00019),
			},
		},
	}
	deposit, isDeposit, err := listener.DecodeDepositEvent(d1, s.mockConn, s.resource)
	s.Equal(isDeposit, true)
	s.NotNil(err)
	s.Equal(deposit, listener.Deposit{})
}

func (s *DecodeEventsSuite) Test_DecodeDepositEvent_ErrorFetchingPrevTx() {
	hash, _ := chainhash.NewHashFromStr("00000000000000000008bba5a6ff31fdb9bb1d4147905b5b3c47a07a07235bfc")

	s.mockConn.EXPECT().GetRawTransactionVerbose(hash).Return(nil, fmt.Errorf("error"))

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
					Type:    "witness_v0_keyhash",
					Address: "tb1qln69zuhdunc9stwfh6t7adexxrcr04ppy6thgm",
				},
				Value: float64(0.00019),
			},
		},
	}
	deposit, isDeposit, err := listener.DecodeDepositEvent(d1, s.mockConn, s.resource)
	s.Equal(isDeposit, true)
	s.NotNil(err)
	s.Equal(deposit, listener.Deposit{})
}

func (s *DecodeEventsSuite) Test_DecodeDepositEvent() {
	hash, _ := chainhash.NewHashFromStr("00000000000000000008bba5a6ff31fdb9bb1d4147905b5b3c47a07a07235bfc")

	result := &btcjson.TxRawResult{
		Vout: []btcjson.Vout{{
			ScriptPubKey: btcjson.ScriptPubKeyResult{
				Address: "tb1qln69zuhdunc9stwfh6t7adexxrcr04ppy6thgm",
			},
		}},
	}
	s.mockConn.EXPECT().GetRawTransactionVerbose(hash).Return(result, nil)

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
					Type:    "witness_v0_keyhash",
					Address: "tb1qln69zuhdunc9stwfh6t7adexxrcr04ppy6thgm",
				},
				Value: float64(0.00019),
			},
		},
	}
	deposit, isDeposit, err := listener.DecodeDepositEvent(d1, s.mockConn, s.resource)
	s.Equal(isDeposit, true)
	s.Nil(err)
	s.Equal(deposit, listener.Deposit{
		ResourceID:    s.resource.ResourceID,
		SenderAddress: d1.Vout[1].ScriptPubKey.Address,
		Amount:        big.NewInt(int64(d1.Vout[1].Value * 1e8)),
		Data:          "0xe9f23A8289764280697a03aC06795eA92a170e42_1",
	})
}

func (s *DecodeEventsSuite) Test_DecodeDepositEvent_NotBridgeDepositTx() {
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
					Type:    "witness_v0_keyhash",
					Address: "NotBridgeAddress",
				},
				Value: float64(0.00019),
			},
		},
	}
	deposit, isDeposit, err := listener.DecodeDepositEvent(d1, s.mockConn, s.resource)
	s.Equal(isDeposit, false)
	s.Nil(err)
	s.Equal(deposit, listener.Deposit{})
}
