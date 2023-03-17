// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package listener

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/rs/zerolog"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/chains/evm/listener"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/consts"
	"github.com/ethereum/go-ethereum/accounts/abi"

	hubEvents "github.com/ChainSafe/sygma-relayer/chains/evm/calls/events"
	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/comm/p2p"
	"github.com/ChainSafe/sygma-relayer/topology"
	"github.com/ChainSafe/sygma-relayer/tss"
	"github.com/ChainSafe/sygma-relayer/tss/keygen"
	"github.com/ChainSafe/sygma-relayer/tss/resharing"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p/core/host"
)

type EventListener interface {
	FetchKeygenEvents(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error)
	FetchRefreshEvents(ctx context.Context, address common.Address, startBlock *big.Int, endBlock *big.Int) ([]*hubEvents.Refresh, error)
	FetchRetryEvents(ctx context.Context, contractAddress common.Address, startBlock *big.Int, endBlock *big.Int) ([]hubEvents.RetryEvent, error)
	FetchDepositEvent(event hubEvents.RetryEvent, bridgeAddress common.Address, blockConfirmations *big.Int) ([]events.Deposit, error)
}

type DepositEventHandler struct {
	log            zerolog.Logger
	eventListener  listener.EventListener
	depositHandler listener.DepositHandler

	bridgeAddress common.Address
	domainID      uint8
}

func NewDepositEventHandler(
	logC zerolog.Context,
	eventListener listener.EventListener,
	depositHandler listener.DepositHandler,
	bridgeAddress common.Address,
	domainID uint8,
) *DepositEventHandler {
	logger := log.With().Str("domainID", domainID).Logger()
	return &DepositEventHandler{
		log:            logger,
		eventListener:  eventListener,
		depositHandler: depositHandler,
		bridgeAddress:  bridgeAddress,
		domainID:       domainID,
	}
}

func (eh *DepositEventHandler) HandleEvent(
	startBlock *big.Int,
	endBlock *big.Int,
	msgChan chan []*message.Message,
) error {
	deposits, err := eh.eventListener.FetchDeposits(context.Background(), eh.bridgeAddress, startBlock, endBlock)
	if err != nil {
		return fmt.Errorf("unable to fetch deposit events because of: %+v", err)
	}

	domainDeposits := make(map[uint8][]*message.Message)
	for _, d := range deposits {
		func(d *events.Deposit) {
			defer func() {
				if r := recover(); r != nil {
					eh.log.Error().Err(err).Msgf("panic occured while handling deposit %+v", d)
				}
			}()

			m, err := eh.depositHandler.HandleDeposit(
				eh.domainID, d.DestinationDomainID, d.DepositNonce, d.ResourceID, d.Data, d.HandlerResponse,
			)
			if err != nil {
				eh.log.Error().Err(err).Str("start block", startBlock.String()).Str(
					"end block", endBlock.String(),
				).Uint8("domainID", eh.domainID).Msgf("%v", err)
				return
			}

			eh.log.Info().Msgf("Resolved deposit message %+v in block range: %s-%s", m, startBlock.String(), endBlock.String())

			if m.Type == PermissionlessGenericTransfer {
				msgChan <- []*message.Message{m}
				return
			}

			domainDeposits[m.Destination] = append(domainDeposits[m.Destination], m)
		}(d)
	}

	for _, deposits := range domainDeposits {
		msgChan <- deposits
	}

	return nil
}

type RetryEventHandler struct {
	log                zerolog.Logger
	eventListener      EventListener
	depositHandler     listener.DepositHandler
	bridgeAddress      common.Address
	bridgeABI          abi.ABI
	domainID           uint8
	blockConfirmations *big.Int
}

func NewRetryEventHandler(
	logC zerolog.Context,
	eventListener EventListener,
	depositHandler listener.DepositHandler,
	bridgeAddress common.Address,
	domainID uint8,
	blockConfirmations *big.Int,
) *RetryEventHandler {
	bridgeABI, _ := abi.JSON(strings.NewReader(consts.BridgeABI))
	logger := log.With().Str("domainID", domainID).Logger()
	return &RetryEventHandler{
		log:                logger,
		eventListener:      eventListener,
		depositHandler:     depositHandler,
		bridgeAddress:      bridgeAddress,
		bridgeABI:          bridgeABI,
		domainID:           domainID,
		blockConfirmations: blockConfirmations,
	}
}

func (eh *RetryEventHandler) HandleEvent(
	startBlock *big.Int,
	endBlock *big.Int,
	msgChan chan []*message.Message,
) error {
	retryEvents, err := eh.eventListener.FetchRetryEvents(context.Background(), eh.bridgeAddress, startBlock, endBlock)
	if err != nil {
		return fmt.Errorf("unable to fetch retry events because of: %+v", err)
	}

	retriesByDomain := make(map[uint8][]*message.Message)
	for _, event := range retryEvents {
		func(event hubEvents.RetryEvent) {
			defer func() {
				if r := recover(); r != nil {
					eh.log.Error().Err(err).Msgf("panic occured while handling retry event %+v", event)
				}
			}()

			deposits, err := eh.eventListener.FetchDepositEvent(event, eh.bridgeAddress, eh.blockConfirmations)
			if err != nil {
				eh.log.Error().Err(err).Msgf("Unable to fetch deposit events from event %+v", event)
				return
			}

			for _, d := range deposits {
				msg, err := eh.depositHandler.HandleDeposit(
					eh.domainID, d.DestinationDomainID, d.DepositNonce,
					d.ResourceID, d.Data, d.HandlerResponse,
				)
				if err != nil {
					eh.log.Error().Err(err).Msgf("Failed handling deposit %+v", d)
					continue
				}

				eh.log.Info().Msgf(
					"Resolved retry message %+v in block range: %s-%s", msg, startBlock.String(), endBlock.String(),
				)
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
	log           zerolog.Logger
	eventListener EventListener
	coordinator   *tss.Coordinator
	host          host.Host
	communication comm.Communication
	storer        keygen.SaveDataStorer
	bridgeAddress common.Address
	threshold     int
}

func NewKeygenEventHandler(
	logC zerolog.Context,
	eventListener EventListener,
	coordinator *tss.Coordinator,
	host host.Host,
	communication comm.Communication,
	storer keygen.SaveDataStorer,
	bridgeAddress common.Address,
	threshold int,
) *KeygenEventHandler {
	return &KeygenEventHandler{
		log:           logC.Logger(),
		eventListener: eventListener,
		coordinator:   coordinator,
		host:          host,
		communication: communication,
		storer:        storer,
		bridgeAddress: bridgeAddress,
		threshold:     threshold,
	}
}

func (eh *KeygenEventHandler) HandleEvent(
	startBlock *big.Int,
	endBlock *big.Int,
	msgChan chan []*message.Message,
) error {
	key, err := eh.storer.GetKeyshare()
	if (key.Threshold != 0) && (err == nil) {
		return nil
	}

	keygenEvents, err := eh.eventListener.FetchKeygenEvents(
		context.Background(), eh.bridgeAddress, startBlock, endBlock,
	)
	if err != nil {
		return fmt.Errorf("unable to fetch keygen events because of: %+v", err)
	}

	if len(keygenEvents) == 0 {
		return nil
	}

	eh.log.Info().Msgf(
		"Resolved keygen message in block range: %s-%s", startBlock.String(), endBlock.String(),
	)

	keygenBlockNumber := big.NewInt(0).SetUint64(keygenEvents[0].BlockNumber)
	keygen := keygen.NewKeygen(eh.sessionID(keygenBlockNumber), eh.threshold, eh.host, eh.communication, eh.storer)
	go eh.coordinator.Execute(context.Background(), keygen, make(chan interface{}, 1), make(chan error, 1))

	return nil
}

func (eh *KeygenEventHandler) sessionID(block *big.Int) string {
	return fmt.Sprintf("keygen-%s", block.String())
}

type RefreshEventHandler struct {
	log              zerolog.Logger
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
	logC zerolog.Context,
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
		log:              logC.Logger(),
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
func (eh *RefreshEventHandler) HandleEvent(
	startBlock *big.Int,
	endBlock *big.Int,
	msgChan chan []*message.Message,
) error {
	refreshEvents, err := eh.eventListener.FetchRefreshEvents(
		context.Background(), eh.bridgeAddress, startBlock, endBlock,
	)
	if err != nil {
		return fmt.Errorf("unable to fetch keygen events because of: %+v", err)
	}
	if len(refreshEvents) == 0 {
		return nil
	}

	hash := refreshEvents[len(refreshEvents)-1].Hash
	if hash == "" {
		return fmt.Errorf("hash cannot be empty string")
	}
	topology, err := eh.topologyProvider.NetworkTopology(hash)
	if err != nil {
		return err
	}
	err = eh.topologyStore.StoreTopology(topology)
	if err != nil {
		return err
	}

	eh.connectionGate.SetTopology(topology)
	p2p.LoadPeers(eh.host, topology.Peers)

	eh.log.Info().Msgf(
		"Resolved refresh message in block range: %s-%s", startBlock.String(), endBlock.String(),
	)

	resharing := resharing.NewResharing(
		eh.sessionID(startBlock), topology.Threshold, eh.host, eh.communication, eh.storer,
	)
	go eh.coordinator.Execute(context.Background(), resharing, make(chan interface{}, 1), make(chan error, 1))

	return nil
}

func (eh *RefreshEventHandler) sessionID(block *big.Int) string {
	return fmt.Sprintf("resharing-%s", block.String())
}
