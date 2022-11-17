package listener

import (
	"errors"
	"math/big"
	"time"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"

	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/centrifuge/go-substrate-rpc-client/types"
	"github.com/rs/zerolog/log"
)

var BlockRetryInterval = time.Second * 5

var ErrBlockNotReady = errors.New("required result to be 32 bytes, but got 0")

type EventHandler interface {
	HandleEvents(evtI interface{}, msgChan chan []*message.Message) error
}
type ChainClient interface {
	GetHeaderLatest() (*types.Header, error)
	GetBlockHash(blockNumber uint64) (types.Hash, error)
	GetBlockEvents(hash types.Hash, target interface{}) error
	UpdateMetatdata() error
}

func NewSubstrateListener(client ChainClient, eventHandlers []EventHandler) *SubstrateListener {
	return &SubstrateListener{
		client:        client,
		eventHandlers: eventHandlers,
	}
}

type SubstrateListener struct {
	client        ChainClient
	eventHandlers []EventHandler
}

func (l *SubstrateListener) ListenToEvents(startBlock *big.Int, domainID uint8, blockstore store.BlockStore, stopChn <-chan struct{}, msgChan chan []*message.Message, errChn chan<- error) {
	go func() {
		for {
			select {
			case <-stopChn:
				return
			default:
				// retrieves the header of the latest block
				finalizedHeader, err := l.client.GetHeaderLatest()
				if err != nil {
					log.Error().Err(err).Msg("Failed to fetch finalized header")
					time.Sleep(BlockRetryInterval)
					continue
				}

				if startBlock == nil {
					startBlock = big.NewInt(int64(finalizedHeader.Number))
				}

				if startBlock.Cmp(big.NewInt(0).SetUint64(uint64(finalizedHeader.Number))) == 1 {
					time.Sleep(BlockRetryInterval)
					continue
				}
				hash, err := l.client.GetBlockHash(startBlock.Uint64())
				if err != nil && err.Error() == ErrBlockNotReady.Error() {
					time.Sleep(BlockRetryInterval)
					continue
				} else if err != nil {
					log.Error().Err(err).Str("block", startBlock.String()).Msg("Failed to query latest block")
					time.Sleep(BlockRetryInterval)
					continue
				}
				evts := &events.Events{}
				err = l.client.GetBlockEvents(hash, evts)
				if err != nil {
					log.Error().Err(err).Msg("Failed to process events in block")
					continue
				}

				for _, handler := range l.eventHandlers {
					err := handler.HandleEvents(evts, msgChan)
					if err != nil {
						log.Error().Err(err).Msg("Error handling substrate events")
						continue
					}
				}
				if len(evts.System_CodeUpdated) > 0 {
					err := l.client.UpdateMetatdata()
					if err != nil {
						log.Error().Err(err).Msg("Unable to update Metadata")
						log.Error().Err(err).Msgf("%v", err)
						return
					}
				}

				if startBlock.Int64()%20 == 0 {
					// Logging process every 20 blocks to exclude spam
					log.Debug().Str("block", startBlock.String()).Uint8("domainID", domainID).Msg("Queried block for deposit events")
				}
				err = blockstore.StoreBlock(startBlock, domainID)
				if err != nil {
					log.Error().Str("block", startBlock.String()).Err(err).Msg("Failed to write latest block to blockstore")
				}
				startBlock.Add(startBlock, big.NewInt(1))
			}
		}
	}()
}
