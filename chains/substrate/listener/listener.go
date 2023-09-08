// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"context"
	"github.com/ChainSafe/chainbridge-core/observability"
	"github.com/status-im/keycard-go/hexutils"
	"go.opentelemetry.io/otel/attribute"
	"math/big"
	"time"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/substrate"

	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/parser"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type EventHandler interface {
	HandleEvents(ctx context.Context, evts []*parser.Event, msgChan chan []*message.Message) error
}
type ChainConnection interface {
	UpdateMetatdata() error
	GetHeaderLatest() (*types.Header, error)
	GetBlockHash(blockNumber uint64) (types.Hash, error)
	GetBlockEvents(hash types.Hash) ([]*parser.Event, error)
	GetFinalizedHead() (types.Hash, error)
	GetBlock(blockHash types.Hash) (*types.SignedBlock, error)
}

type SubstrateListener struct {
	conn ChainConnection

	eventHandlers      []EventHandler
	blockRetryInterval time.Duration
	blockInterval      *big.Int

	log zerolog.Logger
}

func NewSubstrateListener(connection ChainConnection, eventHandlers []EventHandler, config *substrate.SubstrateConfig) *SubstrateListener {
	return &SubstrateListener{
		log:                log.With().Uint8("domainID", *config.GeneralChainConfig.Id).Logger(),
		conn:               connection,
		eventHandlers:      eventHandlers,
		blockRetryInterval: config.BlockRetryInterval,
		blockInterval:      config.BlockInterval,
	}
}

func (l *SubstrateListener) ListenToEvents(ctx context.Context, startBlock *big.Int, domainID uint8, blockstore store.BlockStore, msgChan chan []*message.Message) {
	endBlock := big.NewInt(0)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				ctxWithSpan, span, logger := observability.CreateSpanAndLoggerFromContext(ctx, "relayer-sygma", "relayer.sygma.SubstrateListener.ListenToEvents")
				hash, err := l.conn.GetFinalizedHead()
				if err != nil {
					_ = observability.LogAndRecordError(&logger, span, err, "unable to get GetFinalizedHead")
					span.End()
					time.Sleep(l.blockRetryInterval)
					continue
				}
				head, err := l.conn.GetBlock(hash)
				if err != nil {
					_ = observability.LogAndRecordError(&logger, span, err, "unable to call GetBlock")
					span.End()
					time.Sleep(l.blockRetryInterval)
					continue
				}

				if startBlock == nil {
					startBlock = big.NewInt(int64(head.Block.Header.Number))
				}
				endBlock.Add(startBlock, l.blockInterval)

				observability.SetSpanAndLoggerAttrs(&logger, span, attribute.String("startBlock", startBlock.String()), attribute.String("endBlock", endBlock.String()))

				// Sleep if finalized is less then current block
				if big.NewInt(int64(head.Block.Header.Number)).Cmp(endBlock) == -1 {
					time.Sleep(l.blockRetryInterval)
					observability.LogAndEvent(logger.Debug(), span, "finalized is less then current block")
					span.End()
					continue
				}
				evts, err := l.fetchEvents(ctxWithSpan, startBlock, endBlock)
				if err != nil {
					_ = observability.LogAndRecordError(&logger, span, err, "failed fetching events for block range")
					time.Sleep(l.blockRetryInterval)
					span.End()
					continue
				}

				for _, handler := range l.eventHandlers {
					err := handler.HandleEvents(ctxWithSpan, evts, msgChan)
					if err != nil {
						_ = observability.LogAndRecordError(&logger, span, err, "failed handling substrate events")
						continue
					}
				}
				err = blockstore.StoreBlock(startBlock, domainID)
				if err != nil {
					_ = observability.LogAndRecordError(&logger, span, err, "failed to write latest block to blockstore")
				}
				startBlock.Add(startBlock, l.blockInterval)
				span.End()
			}
		}
	}()
}

func (l *SubstrateListener) fetchEvents(ctx context.Context, startBlock *big.Int, endBlock *big.Int) ([]*parser.Event, error) {
	_, span, logger := observability.CreateSpanAndLoggerFromContext(ctx, "relayer-sygma", "relayer.sygma.SubstrateListener.fetchEvents", attribute.String("startBlock", startBlock.String()), attribute.String("endBlock", endBlock.String()))
	defer span.End()

	evts := make([]*parser.Event, 0)
	for i := new(big.Int).Set(startBlock); i.Cmp(endBlock) == -1; i.Add(i, big.NewInt(1)) {
		hash, err := l.conn.GetBlockHash(i.Uint64())
		if err != nil {
			return nil, observability.LogAndRecordErrorWithStatus(&logger, span, err, "failed to call GetBlockHash")
		}

		evt, err := l.conn.GetBlockEvents(hash)
		if err != nil {
			return nil, observability.LogAndRecordErrorWithStatus(&logger, span, err, "failed to call GetBlockEvents")
		}
		for _, e := range evt {
			observability.LogAndEvent(logger.Debug(), span, "Event fetched", attribute.String("event.name", e.Name), attribute.String("event.ID", hexutils.BytesToHex(e.EventID[:])))
		}
		evts = append(evts, evt...)
	}
	return evts, nil
}
