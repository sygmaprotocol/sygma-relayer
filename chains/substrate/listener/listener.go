// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	traceapi "go.opentelemetry.io/otel/trace"
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
				ctxWithSpan, span := otel.Tracer("relayer-sygma").Start(ctx, "relayer.sygma.SubstrateListener.ListenToEvents")
				hash, err := l.conn.GetFinalizedHead()
				if err != nil {
					l.log.Error().Err(err).Msg("Failed to fetch finalized header")
					time.Sleep(l.blockRetryInterval)
					continue
				}
				head, err := l.conn.GetBlock(hash)
				if err != nil {
					l.log.Error().Err(err).Msg("Failed to fetch block")
					time.Sleep(l.blockRetryInterval)
					continue
				}

				if startBlock == nil {
					startBlock = big.NewInt(int64(head.Block.Header.Number))
				}
				endBlock.Add(startBlock, l.blockInterval)

				// Sleep if finalized is less then current block
				if big.NewInt(int64(head.Block.Header.Number)).Cmp(endBlock) == -1 {
					time.Sleep(l.blockRetryInterval)
					continue
				}
				// This context will share context between fetchEvents and HandleEvents. Might be a better idea to refactor it in future.
				evts, err := l.fetchEvents(ctxWithSpan, startBlock, endBlock)
				if err != nil {
					l.log.Err(err).Msgf("Failed fetching events for block range %s-%s", startBlock, endBlock)
					time.Sleep(l.blockRetryInterval)
					continue
				}

				for _, handler := range l.eventHandlers {
					err := handler.HandleEvents(ctxWithSpan, evts, msgChan)
					if err != nil {
						l.log.Error().Err(err).Msg("Error handling substrate events")
						continue
					}
				}
				err = blockstore.StoreBlock(startBlock, domainID)
				if err != nil {
					l.log.Error().Str("block", startBlock.String()).Err(err).Msg("Failed to write latest block to blockstore")
				}
				startBlock.Add(startBlock, l.blockInterval)
				span.End()
			}
		}
	}()
}

func (l *SubstrateListener) fetchEvents(ctx context.Context, startBlock *big.Int, endBlock *big.Int) ([]*parser.Event, error) {
	_, span := otel.Tracer("relayer-sygma").Start(ctx, "relayer.sygma.SubstrateListener.fetchEvents")
	defer span.End()
	span.SetAttributes(attribute.String("startBlock", startBlock.String()), attribute.String("endBlock", endBlock.String()))

	l.log.Debug().Msgf("Fetching substrate events for block range %s-%s", startBlock, endBlock)

	evts := make([]*parser.Event, 0)
	for i := new(big.Int).Set(startBlock); i.Cmp(endBlock) == -1; i.Add(i, big.NewInt(1)) {
		hash, err := l.conn.GetBlockHash(i.Uint64())
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		evt, err := l.conn.GetBlockEvents(hash)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		for _, e := range evt {
			span.AddEvent("Event fetched", traceapi.WithAttributes(attribute.String("event.name", e.Name)))
		}
		evts = append(evts, evt...)
	}
	span.SetStatus(codes.Ok, "Events fetched")
	return evts, nil
}
