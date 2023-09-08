// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/status-im/keycard-go/hexutils"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/observability"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/parser"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
		if e.Name == events.ParachainUpdatedEvent {
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

func (eh *FungibleTransferEventHandler) HandleEvents(ctx context.Context, evts []*parser.Event, msgChan chan []*message.Message) error {
	_, span, logger := observability.CreateSpanAndLoggerFromContext(ctx, "relayer-sygma", "relayer.sygma.SubstrateFungibleTransferEventHandler.HandleEvents")
	defer span.End()
	domainDeposits := make(map[uint8][]*message.Message)

	eventIDS := make([]string, 0)

	for _, evt := range evts {
		if evt.Name == events.DepositEvent {
			eventIDS = append(eventIDS, hex.EncodeToString(evt.EventID[:]))
			func(evt parser.Event) {
				defer func() {
					if r := recover(); r != nil {
						_ = observability.LogAndRecordError(&logger, span, errors.New("panic"), "failed to  while handle deposit", attribute.String("event.id", hex.EncodeToString(evt.EventID[:])))
					}
				}()

				var d events.Deposit
				err := mapstructure.Decode(evt.Fields, &d)
				if err != nil {
					_ = observability.LogAndRecordError(&logger, span, err, "failed to Decode mapstructure")
					return
				}

				observability.SetSpanAndLoggerAttrs(&logger, span, d.TraceEventAttributes()...)

				m, err := eh.depositHandler.HandleDeposit(eh.domainID, d.DestDomainID, d.DepositNonce, d.ResourceID, d.CallData, d.TransferType)
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					_ = observability.LogAndRecordError(&logger, span, err, "failed to HandleDeposit", attribute.String("msg.id", m.ID()), attribute.String("msg.full", m.String()))
					return
				}

				observability.LogAndEvent(logger.Info(), span, "Resolved deposit message", attribute.String("msg.id", m.ID()), attribute.String("msg.full", m.String()))
				domainDeposits[m.Destination] = append(domainDeposits[m.Destination], m)
			}(*evt)
		}
	}
	span.SetAttributes(attribute.StringSlice("eventIDs", eventIDS))
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

func (rh *RetryEventHandler) HandleEvents(ctx context.Context, evts []*parser.Event, msgChan chan []*message.Message) error {
	_, span, logger := observability.CreateSpanAndLoggerFromContext(ctx, "relayer-sygma", "relayer.sygma.SubstrateRetryEventHandler.HandleEvent")
	defer span.End()
	hash, err := rh.conn.GetFinalizedHead()
	if err != nil {
		return observability.LogAndRecordErrorWithStatus(&logger, span, err, "failed to GetFinalizedHead")
	}
	finalized, err := rh.conn.GetBlock(hash)
	if err != nil {
		return observability.LogAndRecordErrorWithStatus(&logger, span, err, "failed to GetBlock")

	}
	finalizedBlockNumber := big.NewInt(int64(finalized.Block.Header.Number))

	domainDeposits := make(map[uint8][]*message.Message)
	for _, evt := range evts {
		if evt.Name == events.RetryEvent {
			observability.SetSpanAndLoggerAttrs(&logger, span, attribute.String("evt.ID", hexutils.BytesToHex(evt.EventID[:])))
			err := func(evt parser.Event) error {
				defer func() {
					if r := recover(); r != nil {
						logger.Error().Msgf("panic occured while handling retry event %+v because %s", evt, r)
						_ = observability.LogAndRecordError(&logger, span, fmt.Errorf(fmt.Sprintf("%s", r)), "failed to handle retry event")

					}
				}()
				var er events.Retry
				err = mapstructure.Decode(evt.Fields, &er)
				if err != nil {
					return fmt.Errorf("failed to Decode evt %w", err)
				}
				// (latestBlockNumber - event.DepositOnBlockHeight) == blockConfirmations
				if big.NewInt(finalizedBlockNumber.Int64()).Cmp(er.DepositOnBlockHeight.Int) == -1 {
					logger.Debug().Msgf("Retry event for block number %d has not enough confirmations", er.DepositOnBlockHeight)
					return nil
				}

				bh, err := rh.conn.GetBlockHash(er.DepositOnBlockHeight.Uint64())
				if err != nil {
					return fmt.Errorf("failed to GetBlockHash evt %w", err)
				}

				bEvts, err := rh.conn.GetBlockEvents(bh)
				if err != nil {
					return fmt.Errorf("failed to GetBlockEvents evt %w", err)
				}

				for _, event := range bEvts {
					if event.Name == events.DepositEvent {
						var d events.Deposit
						err = mapstructure.Decode(event.Fields, &d)
						if err != nil {
							span.SetStatus(codes.Error, err.Error())
							return err
						}
						m, err := rh.depositHandler.HandleDeposit(rh.domainID, d.DestDomainID, d.DepositNonce, d.ResourceID, d.CallData, d.TransferType)
						if err != nil {
							return fmt.Errorf("failed to HandleDeposit %w", err)
						}
						domainDeposits[m.Destination] = append(domainDeposits[m.Destination], m)
						observability.LogAndEvent(logger.Info(), span, "resolved Deposit to retry", d.TraceEventAttributes()...)
					}
				}
				return nil
			}(*evt)
			if err != nil {
				return observability.LogAndRecordErrorWithStatus(&logger, span, err, "failed to Handle Substrate Retry event")

			}
		}
	}
	for _, deposits := range domainDeposits {
		msgChan <- deposits
	}
	return nil
}
