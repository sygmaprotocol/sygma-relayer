package listener

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/rs/zerolog/log"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/chains/evm/listener"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/consts"

	hubEvents "github.com/ChainSafe/sygma-relayer/chains/evm/calls/events"
	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/comm/p2p"
	"github.com/ChainSafe/sygma-relayer/topology"
	"github.com/ChainSafe/sygma-relayer/tss"
	"github.com/ChainSafe/sygma-relayer/tss/keygen"
	"github.com/ChainSafe/sygma-relayer/tss/resharing"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p-core/host"
)

type EventListener interface {
	FetchKeygenEvents(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error)
	FetchRefreshEvents(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]*hubEvents.Refresh, error)
	FetchRetryEvents(ctx context.Context, contractAddress common.Address, startBlock *big.Int, endBlock *big.Int) ([]hubEvents.RetryEvent, error)
	FetchDepositEvent(event hubEvents.RetryEvent, bridgeAddress common.Address, blockConfirmations *big.Int) ([]events.Deposit, error)
}

type RetryEventHandler struct {
	eventListener      EventListener
	depositHandler     listener.DepositHandler
	bridgeAddress      common.Address
	bridgeABI          abi.ABI
	domainID           uint8
	blockConfirmations *big.Int
}

func NewRetryEventHandler(
	eventListener EventListener,
	depositHandler listener.DepositHandler,
	bridgeAddress common.Address,
	domainID uint8,
	blockConfirmations *big.Int,
) *RetryEventHandler {
	bridgeABI, _ := abi.JSON(strings.NewReader(consts.BridgeABI))
	return &RetryEventHandler{
		eventListener:      eventListener,
		depositHandler:     depositHandler,
		bridgeAddress:      bridgeAddress,
		bridgeABI:          bridgeABI,
		domainID:           domainID,
		blockConfirmations: blockConfirmations,
	}
}

func (eh *RetryEventHandler) HandleEvent(startBlock *big.Int, endBlock *big.Int, msgChan chan []*message.Message) error {
	retryEvents, err := eh.eventListener.FetchRetryEvents(context.Background(), eh.bridgeAddress, startBlock, endBlock)
	if err != nil {
		return fmt.Errorf("unable to fetch retry events because of: %+v", err)
	}

	retriesByDomain := make(map[uint8][]*message.Message)
	for _, event := range retryEvents {
		func(event hubEvents.RetryEvent) {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Err(err).Msgf("panic occured while handling retry event %+v", event)
				}
			}()

			deposits, err := eh.eventListener.FetchDepositEvent(event, eh.bridgeAddress, eh.blockConfirmations)
			if err != nil {
				log.Error().Err(err).Msgf("Unable to fetch deposit events from event %+v", event)
				return
			}

			for _, d := range deposits {
				msg, err := eh.depositHandler.HandleDeposit(
					eh.domainID, d.DestinationDomainID, d.DepositNonce,
					d.ResourceID, d.Data, d.HandlerResponse,
				)
				if err != nil {
					log.Error().Err(err).Msgf("Failed handling deposit %+v", d)
					continue
				}

				log.Debug().Msgf("Resolved retry message %+v in block range: %s-%s", msg, startBlock.String(), endBlock.String())
				retriesByDomain[msg.Destination] = append(retriesByDomain[msg.Destination], msg)
			}
		}(event)
	}

	for _, retries := range retriesByDomain {
		msgChan <- retries
	}

	return nil
}

type KeygenEventHandler struct {
	eventListener EventListener
	coordinator   *tss.Coordinator
	host          host.Host
	communication comm.Communication
	storer        keygen.SaveDataStorer
	bridgeAddress common.Address
	threshold     int
}

func NewKeygenEventHandler(
	eventListener EventListener,
	coordinator *tss.Coordinator,
	host host.Host,
	communication comm.Communication,
	storer keygen.SaveDataStorer,
	bridgeAddress common.Address,
	threshold int,
) *KeygenEventHandler {
	return &KeygenEventHandler{
		eventListener: eventListener,
		coordinator:   coordinator,
		host:          host,
		communication: communication,
		storer:        storer,
		bridgeAddress: bridgeAddress,
		threshold:     threshold,
	}
}

func (eh *KeygenEventHandler) HandleEvent(startBlock *big.Int, endBlock *big.Int, msgChan chan []*message.Message) error {
	key, err := eh.storer.GetKeyshare()
	if (key.Threshold != 0) && (err == nil) {
		return nil
	}

	keygenEvents, err := eh.eventListener.FetchKeygenEvents(context.Background(), eh.bridgeAddress, startBlock, endBlock)
	if err != nil {
		return fmt.Errorf("unable to fetch keygen events because of: %+v", err)
	}
	if len(keygenEvents) == 0 {
		return nil
	}

	keygenBlockNumber := big.NewInt(0).SetUint64(keygenEvents[0].BlockNumber)
	keygen := keygen.NewKeygen(eh.sessionID(keygenBlockNumber), eh.threshold, eh.host, eh.communication, eh.storer)
	go eh.coordinator.Execute(context.Background(), keygen, make(chan interface{}, 1), make(chan error, 1))

	return nil
}

func (eh *KeygenEventHandler) sessionID(block *big.Int) string {
	return fmt.Sprintf("keygen-%s", block.String())
}

type RefreshEventHandler struct {
	topologyProvider topology.NetworkTopologyProvider
	topologyStore    *topology.TopologyStore
	eventListener    EventListener
	bridgeAddress    common.Address
	coordinator      *tss.Coordinator
	host             host.Host
	communication    comm.Communication
	connectionGate   *p2p.ConnectionGate
	storer           resharing.SaveDataStorer
}

func NewRefreshEventHandler(
	topologyProvider topology.NetworkTopologyProvider,
	topologyStore *topology.TopologyStore,
	eventListener EventListener,
	coordinator *tss.Coordinator,
	host host.Host,
	communication comm.Communication,
	connectionGate *p2p.ConnectionGate,
	storer resharing.SaveDataStorer,
	bridgeAddress common.Address,
) *RefreshEventHandler {
	return &RefreshEventHandler{
		topologyProvider: topologyProvider,
		topologyStore:    topologyStore,
		eventListener:    eventListener,
		coordinator:      coordinator,
		host:             host,
		communication:    communication,
		storer:           storer,
		connectionGate:   connectionGate,
		bridgeAddress:    bridgeAddress,
	}
}

// HandleEvent fetches refresh events and in case of an event retrieves and stores the latest topology
// and starts a resharing tss process
func (eh *RefreshEventHandler) HandleEvent(startBlock *big.Int, endBlock *big.Int, msgChan chan []*message.Message) error {
	refreshEvents, err := eh.eventListener.FetchRefreshEvents(context.Background(), eh.bridgeAddress, startBlock, endBlock)
	if err != nil {
		return fmt.Errorf("unable to fetch keygen events because of: %+v", err)
	}
	if len(refreshEvents) == 0 {
		return nil
	}

	topology, err := eh.topologyProvider.NetworkTopology()
	if err != nil {
		return err
	}
	hash, err := topology.Hash()
	if err != nil {
		return err
	}
	err = eh.topologyStore.StoreTopology(topology)
	if err != nil {
		return err
	}

	// if multiple refresh events inside block range use latest
	expectedHash := refreshEvents[len(refreshEvents)-1].Hash
	if hash != expectedHash {
		return fmt.Errorf("aborting refresh because expected hash %s doesn't match %s", expectedHash, hash)
	}
	eh.connectionGate.SetTopology(topology)
	p2p.LoadPeers(eh.host, topology.Peers)

	resharing := resharing.NewResharing(eh.sessionID(startBlock), topology.Threshold, eh.host, eh.communication, eh.storer)
	go eh.coordinator.Execute(context.Background(), resharing, make(chan interface{}, 1), make(chan error, 1))

	return nil
}

func (eh *RefreshEventHandler) sessionID(block *big.Int) string {
	return fmt.Sprintf("resharing-%s", block.String())
}
