package listener

import (
	"context"
	"fmt"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/chains/evm/listener"
	"github.com/ChainSafe/chainbridge-hub/chains/evm/calls/consts"
	hubEvents "github.com/ChainSafe/chainbridge-hub/chains/evm/calls/events"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
	"strings"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
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
	FetchRefreshEvents(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error)
	FetchRetryEvents(ctx context.Context, contractAddress common.Address, startBlock *big.Int, endBlock *big.Int) ([]hubEvents.RetryEvent, error)
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

func (eh *RetryEventHandler) HandleEvent(block *big.Int, msgChan chan *message.Message) error {
	retryEvents, err := eh.eventListener.FetchRetryEvents(context.Background(), eh.bridgeAddress, block, block)
	if err != nil {
		return fmt.Errorf("unable to fetch retry events because of: %+v", err)
	}
	if len(retryEvents) == 0 {
		return nil
	}

	for _, event := range retryEvents {

		retryDepositTxHash := common.HexToHash(retryEvent.TxHash)
		receipt, err := eh.client.WaitAndReturnTxReceipt(retryDepositTxHash)
		if err != nil {
			return fmt.Errorf(
				"unable to fetch logs for retried deposit %s, because of: %+v", retryDepositTxHash.Hex(), err,
			)
		}

		// find deposit event
		var depositEvent events.Deposit
		for _, log := range receipt.Logs {
			err := eh.bridgeABI.UnpackIntoInterface(depositEvent, "Deposit", log.Data)
			if err == nil {
				break
			}
		}

		msg, err := eh.depositHandler.HandleDeposit(
			eh.domainID, depositEvent.DestinationDomainID, depositEvent.DepositNonce,
			depositEvent.ResourceID, depositEvent.Data, depositEvent.HandlerResponse,
		)
		if err != nil {
			return err
		}

		msgChan <- msg
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

func (eh *KeygenEventHandler) HandleEvent(block *big.Int, msgChan chan *message.Message) error {
	keygenEvents, err := eh.eventListener.FetchKeygenEvents(context.Background(), eh.bridgeAddress, block, block)
	if err != nil {
		return fmt.Errorf("unable to fetch keygen events because of: %+v", err)
	}
	if len(keygenEvents) == 0 {
		return nil
	}

	keygen := keygen.NewKeygen(eh.sessionID(block), eh.threshold, eh.host, eh.communication, eh.storer)
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

func (eh *RefreshEventHandler) HandleEvent(block *big.Int, msgChan chan *message.Message) error {
	refreshEvents, err := eh.eventListener.FetchRefreshEvents(context.Background(), eh.bridgeAddress, block, block)
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
	p2p.LoadPeers(eh.host, topology.Peers)

	resharing := resharing.NewResharing(eh.sessionID(block), eh.threshold, eh.host, eh.communication, eh.storer)
	go eh.coordinator.Execute(context.Background(), resharing, make(chan interface{}, 1), make(chan error, 1))

	return nil
}

func (eh *RefreshEventHandler) sessionID(block *big.Int) string {
	return fmt.Sprintf("resharing-%s", block.String())
}
