// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"math/big"
	"strconv"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type Deposit struct {
	// ResourceID used to find address of handler to be used for deposit
	ResourceID [32]byte
	// Address of sender (msg.sender: user)
	SenderAddress string
	// Additional data to be passed to specified handler
	Amount *big.Int
	Data   string
}

type DepositHandler interface {
	HandleDeposit(
		sourceID uint8,
		depositNonce uint64,
		resourceID [32]byte,
		amount *big.Int,
		data string,
		blockNumber *big.Int,
	) (*message.Message, error)
}

type Connection interface {
	FetchEvents(startBlock, endBlock *big.Int) ([]btcjson.TxRawResult, error)
	GetRawTransactionVerbose(*chainhash.Hash) (*btcjson.TxRawResult, error)
}

type FungibleTransferEventHandler struct {
	depositHandler DepositHandler
	domainID       uint8
	log            zerolog.Logger
	conn           *rpcclient.Client
	msgChan        chan []*message.Message
	bridge         string
}

func NewFungibleTransferEventHandler(logC zerolog.Context, domainID uint8, depositHandler DepositHandler, msgChan chan []*message.Message, conn *rpcclient.Client, bridge string) *FungibleTransferEventHandler {
	return &FungibleTransferEventHandler{
		depositHandler: depositHandler,
		domainID:       domainID,
		log:            logC.Logger(),
		conn:           conn,
		msgChan:        msgChan,
		bridge:         bridge,
	}
}

func (eh *FungibleTransferEventHandler) HandleEvents(blockNumber *big.Int) error {
	domainDeposits := make(map[uint8][]*message.Message)
	evts, err := eh.FetchEvents(blockNumber)
	if err != nil {
		eh.log.Error().Err(err).Msg("Error fetching events")
		return err
	}
	for evtNumber, evt := range evts {
		func(evt btcjson.TxRawResult) {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msgf("panic occured while handling deposit %+v", evt)
				}
			}()

			d, isDeposit, err := DecodeDepositEvent(evt, eh.conn, eh.bridge)
			if err != nil {
				log.Error().Err(err).Msgf("%v", err)
				return
			}
			if !isDeposit {
				return
			}

			nonce, err := eh.getNonce(blockNumber, evtNumber)
			if err != nil {
				log.Error().Err(err).Msgf("%v", err)
				return
			}
			m, err := eh.depositHandler.HandleDeposit(eh.domainID, nonce, d.ResourceID, d.Amount, d.Data, blockNumber)
			if err != nil {
				log.Error().Err(err).Msgf("%v", err)
				return
			}

			log.Debug().Str("messageID", m.ID).Msgf("Resolved message %+v in block: %s", m, blockNumber.String())
			domainDeposits[m.Destination] = append(domainDeposits[m.Destination], m)
		}(evt)
	}

	for _, deposits := range domainDeposits {
		go func(d []*message.Message) {
			eh.msgChan <- d
		}(deposits)
	}
	return nil
}

func (eh *FungibleTransferEventHandler) FetchEvents(startBlock *big.Int) ([]btcjson.TxRawResult, error) {
	blockHash, err := eh.conn.GetBlockHash(startBlock.Int64())
	if err != nil {
		return nil, err
	}

	// Fetch block details in verbose mode
	block, err := eh.conn.GetBlockVerboseTx(blockHash)
	if err != nil {
		return nil, err
	}
	return block.Tx, nil
}

func (eh *FungibleTransferEventHandler) getNonce(blockNumber *big.Int, evtNumber int) (uint64, error) {

	// Convert blockNumber to string
	blockNumberStr := blockNumber.String()

	// Convert evtNumber to *big.Int
	evtNumberBigInt := big.NewInt(int64(evtNumber))

	// Convert evtNumberBigInt to string
	evtNumberStr := evtNumberBigInt.String()

	// Concatenate blockNumberStr and evtNumberStr
	concatenatedStr := blockNumberStr + evtNumberStr

	// Parse the concatenated string to uint64
	result, err := strconv.ParseUint(concatenatedStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return result, nil
}
