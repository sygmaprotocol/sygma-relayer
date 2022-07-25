package listener

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/rs/zerolog/log"

	"github.com/ChainSafe/sygma-core/chains/evm/calls"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/events"
	"github.com/ChainSafe/sygma-core/chains/evm/listener"
	"github.com/ChainSafe/sygma-core/relayer/message"
	"github.com/ChainSafe/sygma/chains/evm/calls/consts"

	hubEvents "github.com/ChainSafe/sygma/chains/evm/calls/events"
	"github.com/ChainSafe/sygma/comm"
	"github.com/ChainSafe/sygma/comm/p2p"
	"github.com/ChainSafe/sygma/topology"
	"github.com/ChainSafe/sygma/tss"
	"github.com/ChainSafe/sygma/tss/keygen"
	"github.com/ChainSafe/sygma/tss/resharing"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p-core/host"
)

type EventListener interface {
	FetchKeygenEvents(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error)
	FetchRefreshEvents(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]*hubEvents.Refresh, error)
	FetchRetryEvents(ctx context.Context, contractAddress common.Address, startBlock *big.Int, endBlock *big.Int) ([]hubEvents.RetryEvent, error)
	FetchDepositEvent(event hubEvents.RetryEvent) (events.Deposit, error)
}

type RetryEventHandler struct {
	client         calls.ClientDispatcher
	eventListener  EventListener
	depositHandler listener.DepositHandler
	bridgeAddress  common.Address
	bridgeABI      abi.ABI
	domainID       uint8
}

func NewRetryEventHandler(
	client calls.ClientDispatcher,
	eventListener EventListener,
	depositHandler listener.DepositHandler,
	bridgeAddress common.Address,
	domainID uint8,
) *RetryEventHandler {
	bridgeABI, _ := abi.JSON(strings.NewReader(consts.BridgeABI))
	return &RetryEventHandler{
		eventListener:  eventListener,
		depositHandler: depositHandler,
		bridgeAddress:  bridgeAddress,
		client:         client,
		bridgeABI:      bridgeABI,
		domainID:       domainID,
	}
}

func (eh *RetryEventHandler) HandleEvent(startBlock *big.Int, endBlock *big.Int, msgChan chan []*message.Message) error {
	retryEvents, err := eh.eventListener.FetchRetryEvents(context.Background(), eh.bridgeAddress, startBlock, endBlock)
	if err != nil {
		return fmt.Errorf("unable to fetch retry events because of: %+v", err)
	}
	if len(retryEvents) == 0 {
		return nil
	}

	retriesByDomain := make(map[uint8][]*message.Message)
	for _, event := range retryEvents {
		depositEvent, err := eh.eventListener.FetchDepositEvent(event)
		if err != nil {
			return err
		}
		msg, err := eh.depositHandler.HandleDeposit(
			eh.domainID, depositEvent.DestinationDomainID, depositEvent.DepositNonce,
			depositEvent.ResourceID, depositEvent.Data, depositEvent.HandlerResponse,
		)
		if err != nil {
			log.Error().Err(err).Str("start block", startBlock.String()).Str("end block", endBlock.String()).Uint8("domainID", eh.domainID).Msgf("%v", err)
			continue
		}

		log.Debug().Msgf("Resolved retry message %+v in block range: %s-%s", msg, startBlock.String(), endBlock.String())
		retriesByDomain[msg.Destination] = append(retriesByDomain[msg.Destination], msg)
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

	keygen := keygen.NewKeygen(eh.sessionID(startBlock), eh.threshold, eh.host, eh.communication, eh.storer)
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
	storer           resharing.SaveDataStorer
}

func NewRefreshEventHandler(
	topologyProvider topology.NetworkTopologyProvider,
	topologyStore *topology.TopologyStore,
	eventListener EventListener,
	coordinator *tss.Coordinator,
	host host.Host,
	communication comm.Communication,
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
	p2p.LoadPeers(eh.host, topology.Peers)

	resharing := resharing.NewResharing(eh.sessionID(startBlock), topology.Threshold, eh.host, eh.communication, eh.storer)
	go eh.coordinator.Execute(context.Background(), resharing, make(chan interface{}, 1), make(chan error, 1))

	return nil
}

func (eh *RefreshEventHandler) sessionID(block *big.Int) string {
	return fmt.Sprintf("resharing-%s", block.String())
}
