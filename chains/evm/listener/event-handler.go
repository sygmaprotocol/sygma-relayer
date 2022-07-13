package listener

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-hub/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-hub/comm"
	"github.com/ChainSafe/chainbridge-hub/comm/p2p"
	"github.com/ChainSafe/chainbridge-hub/topology"
	"github.com/ChainSafe/chainbridge-hub/tss"
	"github.com/ChainSafe/chainbridge-hub/tss/keygen"
	"github.com/ChainSafe/chainbridge-hub/tss/resharing"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p-core/host"
)

type EventListener interface {
	FetchKeygenEvents(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error)
	FetchRefreshEvents(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]*events.Refresh, error)
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
	eventListener    EventListener
	bridgeAddress    common.Address
	coordinator      *tss.Coordinator
	host             host.Host
	communication    comm.Communication
	storer           resharing.SaveDataStorer
	threshold        int
}

func NewRefreshEventHandler(
	topologyProvider topology.NetworkTopologyProvider,
	eventListener EventListener,
	coordinator *tss.Coordinator,
	host host.Host,
	communication comm.Communication,
	storer resharing.SaveDataStorer,
	bridgeAddress common.Address,
	threshold int,
) *RefreshEventHandler {
	return &RefreshEventHandler{
		topologyProvider: topologyProvider,
		eventListener:    eventListener,
		coordinator:      coordinator,
		host:             host,
		communication:    communication,
		storer:           storer,
		bridgeAddress:    bridgeAddress,
		threshold:        threshold,
	}
}

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
	expectedHash := refreshEvents[len(refreshEvents)-1].Hash
	// if multiple refresh events inside block range use latest
	if hash != expectedHash {
		return fmt.Errorf("aborting refresh because expected hash %s doesn't match %s", expectedHash, hash)
	}
	p2p.LoadPeers(eh.host, topology.Peers)

	resharing := resharing.NewResharing(eh.sessionID(startBlock), eh.threshold, eh.host, eh.communication, eh.storer)
	go eh.coordinator.Execute(context.Background(), resharing, make(chan interface{}, 1), make(chan error, 1))

	return nil
}

func (eh *RefreshEventHandler) sessionID(block *big.Int) string {
	return fmt.Sprintf("resharing-%s", block.String())
}
