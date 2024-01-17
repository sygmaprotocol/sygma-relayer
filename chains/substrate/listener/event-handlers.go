// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"math/big"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/parser"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type SystemUpdateEventHandler struct {
	conn ChainConnection
}

func NewSystemUpdateEventHandler(conn ChainConnection) *SystemUpdateEventHandler {
	return &SystemUpdateEventHandler{
		conn: conn,
	}
}

func (eh *SystemUpdateEventHandler) HandleEvents(evts []*parser.Event) error {
	for _, e := range evts {
		if e.Name == events.ParachainUpdatedEvent {
			log.Info().Msgf("Updating substrate metadata")

			err := eh.conn.UpdateMetadata()
			if err != nil {
				log.Error().Err(err).Msg("Unable to update Metadata")
				return err
			}
		}
	}

	return nil
}

func DecodeEventToDeposit(evtFields registry.DecodedFields) (events.Deposit, error) {
	var d events.Deposit

	for _, evtField := range evtFields {
		switch evtField.Name {
		case "dest_domain_id":
			err := mapstructure.Decode(evtField.Value, &d.DestDomainID)
			if err != nil {
				return events.Deposit{}, err
			}
		case "resource_id":
			err := mapstructure.Decode(evtField.Value, &d.ResourceID)
			if err != nil {
				return events.Deposit{}, err
			}
		case "deposit_nonce":
			err := mapstructure.Decode(evtField.Value, &d.DepositNonce)
			if err != nil {
				return events.Deposit{}, err
			}
		case "sygma_traits_TransferType":
			err := mapstructure.Decode(evtField.Value, &d.TransferType)
			if err != nil {
				return events.Deposit{}, err
			}
		case "deposit_data":
			err := mapstructure.Decode(evtField.Value, &d.CallData)
			if err != nil {
				return events.Deposit{}, err
			}
		case "handler_response":
			err := mapstructure.Decode(evtField.Value, &d.Handler)
			if err != nil {
				return events.Deposit{}, err
			}
		}
	}
	return d, nil
}

func DecodeEventToRetry(evtFields registry.DecodedFields) (events.Retry, error) {
	var er events.Retry

	for _, evtField := range evtFields {
		switch evtField.Name {
		case "deposit_on_block_height":
			err := mapstructure.Decode(evtField.Value, &er.DepositOnBlockHeight)
			if err != nil {
				return events.Retry{}, err
			}
		case "dest_domain_id":
			err := mapstructure.Decode(evtField.Value, &er.DestDomainID)
			if err != nil {
				return events.Retry{}, err
			}
		}
	}

	return er, nil
}

type DepositHandler interface {
	HandleDeposit(sourceID uint8, destID types.U8, nonce types.U64, resourceID types.Bytes32, calldata []byte, transferType types.U8) (*message.Message, error)
}

type FungibleTransferEventHandler struct {
	domainID       uint8
	depositHandler DepositHandler
	log            zerolog.Logger
	msgChan        chan []*message.Message
}

func NewFungibleTransferEventHandler(logC zerolog.Context, domainID uint8, depositHandler DepositHandler) *FungibleTransferEventHandler {
	return &FungibleTransferEventHandler{
		depositHandler: depositHandler,
		domainID:       domainID,
		log:            logC.Logger(),
	}
}

func (eh *FungibleTransferEventHandler) HandleEvents(evts []*parser.Event) error {
	domainDeposits := make(map[uint8][]*message.Message)

	for _, evt := range evts {
		if evt.Name == events.DepositEvent {
			func(evt parser.Event) {
				defer func() {
					if r := recover(); r != nil {
						log.Error().Msgf("panic occured while handling deposit %+v", evt)
					}
				}()
				d, err := DecodeEventToDeposit(evt.Fields)
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
			eh.msgChan <- d
		}(deposits)
	}
	return nil
}

type RetryEventHandler struct {
	conn           ChainConnection
	domainID       uint8
	depositHandler DepositHandler
	log            zerolog.Logger
	msgChan        chan []*message.Message
}

func NewRetryEventHandler(logC zerolog.Context, conn ChainConnection, depositHandler DepositHandler, domainID uint8) *RetryEventHandler {
	return &RetryEventHandler{
		depositHandler: depositHandler,
		domainID:       domainID,
		conn:           conn,
		log:            logC.Logger(),
	}
}

func (rh *RetryEventHandler) HandleEvents(evts []*parser.Event) error {
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
				er, err := DecodeEventToRetry(evt.Fields)
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
						d, err := DecodeEventToDeposit(event.Fields)
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
		rh.msgChan <- deposits
	}
	return nil
}
