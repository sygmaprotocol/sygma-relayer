// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/parser"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type SystemUpdateEventHandler struct {
	conn ChainConnection
}

func NewSystemUpdateEventHandler(conn ChainConnection) *SystemUpdateEventHandler {
	return &SystemUpdateEventHandler{
		conn: conn,
	}
}

func (eh *SystemUpdateEventHandler) HandleEvents(evts []*parser.Event, msgChan chan []*message.Message) error {
	for _, e := range evts {
		if e.Name == events.CouncilExecutedEvent {
			log.Info().Msgf("Updating substrate metadata")

			err := eh.conn.UpdateMetatdata()
			if err != nil {
				log.Error().Err(err).Msg("Unable to update Metadata")
				return err
			}
		}
	}

	return nil
}

type DepositHandler interface {
	HandleDeposit(sourceID uint8, destID types.U8, nonce types.U64, resourceID types.Bytes32, calldata []byte, transferType types.U8) (*message.Message, error)
}

type FungibleTransferEventHandler struct {
	domainID       uint8
	depositHandler DepositHandler
	log            zerolog.Logger
}

func NewFungibleTransferEventHandler(logC zerolog.Context, domainID uint8, depositHandler DepositHandler) *FungibleTransferEventHandler {
	return &FungibleTransferEventHandler{
		depositHandler: depositHandler,
		domainID:       domainID,
		log:            logC.Logger(),
	}
}

func (eh *FungibleTransferEventHandler) HandleEvents(evts []*parser.Event, msgChan chan []*message.Message) error {
	domainDeposits := make(map[uint8][]*message.Message)

	for _, evt := range evts {
		if evt.Name == events.DepositEvent {
			func(evt parser.Event) {
				defer func() {
					if r := recover(); r != nil {
						log.Error().Msgf("panic occured while handling deposit %+v", evt)
					}
				}()

				var d events.Deposit
				err := mapstructure.Decode(evt.Fields, &d)
				if err != nil {
					log.Error().Err(err).Msgf("%v", err)
					return
				}

				m, err := eh.depositHandler.HandleDeposit(eh.domainID, d.DestDomainID, d.DepositNonce, d.ResourceID, d.CallData, d.TransferType)
				if err != nil {
					log.Error().Err(err).Msgf("%v", err)
					return
				}

				eh.log.Info().Msgf("Resolved deposit message %+v", d)

				domainDeposits[m.Destination] = append(domainDeposits[m.Destination], m)
			}(*evt)
		}
	}

	for _, deposits := range domainDeposits {
		go func(d []*message.Message) {
			msgChan <- d
		}(deposits)
	}
	return nil
}

type RetryEventHandler struct {
	conn           ChainConnection
	domainID       uint8
	depositHandler DepositHandler
	log            zerolog.Logger
}

func NewRetryEventHandler(logC zerolog.Context, conn ChainConnection, depositHandler DepositHandler, domainID uint8) *RetryEventHandler {
	return &RetryEventHandler{
		depositHandler: depositHandler,
		domainID:       domainID,
		conn:           conn,
		log:            logC.Logger(),
	}
}

func (rh *RetryEventHandler) HandleEvents(evts []*parser.Event, msgChan chan []*message.Message) error {
	hash, err := rh.conn.GetFinalizedHead()
	if err != nil {
		return err
	}
	finalized, err := rh.conn.GetBlock(hash)
	if err != nil {
		return err
	}
	finalizedBlockNumber := big.NewInt(int64(finalized.Block.Header.Number))

	domainDeposits := make(map[uint8][]*message.Message)
	for _, evt := range evts {
		if evt.Name == events.RetryEvent {
			err := func(evt parser.Event) error {
				defer func() {
					if r := recover(); r != nil {
						log.Error().Msgf("panic occured while handling retry event %+v because %s", evt, r)
					}
				}()
				var er events.Retry
				err = mapstructure.Decode(evt.Fields, &er)
				if err != nil {
					return err
				}
				// (latestBlockNumber - event.DepositOnBlockHeight) == blockConfirmations
				if big.NewInt(finalizedBlockNumber.Int64()).Cmp(er.DepositOnBlockHeight.Int) == -1 {
					log.Warn().Msgf("Retry event for block number %d has not enough confirmations", er.DepositOnBlockHeight)
					return nil
				}

				bh, err := rh.conn.GetBlockHash(er.DepositOnBlockHeight.Uint64())
				if err != nil {
					return err
				}

				bEvts, err := rh.conn.GetBlockEvents(bh)
				if err != nil {
					return err
				}

				for _, event := range bEvts {
					if event.Name == events.DepositEvent {
						var d events.Deposit
						err = mapstructure.Decode(event.Fields, &d)
						if err != nil {
							return err
						}
						m, err := rh.depositHandler.HandleDeposit(rh.domainID, d.DestDomainID, d.DepositNonce, d.ResourceID, d.CallData, d.TransferType)
						if err != nil {
							return err
						}

						rh.log.Info().Msgf("Resolved retry message %+v", d)

						domainDeposits[m.Destination] = append(domainDeposits[m.Destination], m)
					}
				}

				return nil
			}(*evt)
			if err != nil {
				return err
			}
		}
	}

	for _, deposits := range domainDeposits {
		msgChan <- deposits
	}
	return nil
}
