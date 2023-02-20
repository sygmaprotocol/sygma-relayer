package listener

import (
	"context"
	"math/big"
	"time"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/substrate"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"

	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/rs/zerolog/log"
)

type EventHandler interface {
	HandleEvents(evts []*events.Events, msgChan chan []*message.Message) error
}
type ChainConnection interface {
	GetHeaderLatest() (*types.Header, error)
	GetBlockHash(blockNumber uint64) (types.Hash, error)
	GetBlockEvents(hash types.Hash) (*events.Events, error)
}

func NewSubstrateListener(connection ChainConnection, eventHandlers []EventHandler, config *substrate.SubstrateConfig) *SubstrateListener {
	return &SubstrateListener{
		conn:               connection,
		eventHandlers:      eventHandlers,
		blockRetryInterval: config.BlockRetryInterval,
		blockInterval:      config.BlockInterval,
		blockConfirmations: config.BlockConfirmations,
	}
}

type SubstrateListener struct {
	conn               ChainConnection
	eventHandlers      []EventHandler
	blockRetryInterval time.Duration
	blockInterval      *big.Int
	blockConfirmations *big.Int
}

func (l *SubstrateListener) ListenToEvents(ctx context.Context, startBlock *big.Int, domainID uint8, blockstore store.BlockStore, msgChan chan []*message.Message) {
	endBlock := big.NewInt(0)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				head, err := l.conn.GetHeaderLatest()
				if err != nil {
					log.Error().Err(err).Msg("Failed to fetch finalized header")
					time.Sleep(l.blockRetryInterval)
					continue
				}

				if startBlock == nil {
					startBlock = big.NewInt(int64(head.Number))
				}
				endBlock.Add(startBlock, l.blockInterval)

				// Sleep if the difference is less than needed block confirmations; (latest - current) < BlockDelay
				if new(big.Int).Sub(new(big.Int).SetInt64(int64(head.Number)), endBlock).Cmp(l.blockConfirmations) == -1 {
					time.Sleep(l.blockRetryInterval)
					continue
				}

				evts, err := l.fetchEvents(startBlock, endBlock)
				if err != nil {
					time.Sleep(l.blockRetryInterval)
					continue
				}

				for _, handler := range l.eventHandlers {
					err := handler.HandleEvents(evts, msgChan)
					if err != nil {
						log.Error().Err(err).Msg("Error handling substrate events")
						continue
					}
				}
				err = blockstore.StoreBlock(startBlock, domainID)
				if err != nil {
					log.Error().Str("block", startBlock.String()).Err(err).Msg("Failed to write latest block to blockstore")
				}
				startBlock.Add(startBlock, l.blockInterval)
			}
		}
	}()
}

func (l *SubstrateListener) fetchEvents(startBlock *big.Int, endBlock *big.Int) ([]*events.Events, error) {
	evts := make([]*events.Events, 0)
	for ; startBlock.Cmp(endBlock) == -1; startBlock.Add(startBlock, big.NewInt(1)) {
		hash, err := l.conn.GetBlockHash(startBlock.Uint64())
		if err != nil {
			return nil, err
		}

		evt, err := l.conn.GetBlockEvents(hash)
		if err != nil {
			return nil, err
		}

		evts = append(evts, evt)
	}

	return evts, nil
}
