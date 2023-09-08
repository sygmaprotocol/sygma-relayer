// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/ChainSafe/chainbridge-core/observability"
	"math/big"
	"strings"

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
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	return &DepositEventHandler{
		log:            logC.Logger(),
		eventListener:  eventListener,
		depositHandler: depositHandler,
		bridgeAddress:  bridgeAddress,
		domainID:       domainID,
	}
}

func (eh *DepositEventHandler) HandleEvent(
	ctx context.Context,
	startBlock *big.Int,
	endBlock *big.Int,
	msgChan chan []*message.Message,
) error {
	ctx, span, logger := observability.CreateSpanAndLoggerFromContext(ctx, "relayer-sygma", "relayer.sygma.DepositEventHandler.HandleEvent", attribute.String("startBlock", startBlock.String()), attribute.String("endBlock", endBlock.String()))
	defer span.End()
	deposits, err := eh.eventListener.FetchDeposits(ctx, eh.bridgeAddress, startBlock, endBlock)
	if err != nil {
		return observability.LogAndRecordErrorWithStatus(nil, span, err, "unable to fetch deposit events")
	}
	domainDeposits := make(map[uint8][]*message.Message)
	for _, d := range deposits {
		func(d *events.Deposit) {
			defer func() {
				if r := recover(); r != nil {
					_ = observability.LogAndRecordError(&logger, span, errors.New("panic"), "failed to  while handle deposit", d.TraceEventAttributes()...)
				}
			}()
			m, err := eh.depositHandler.HandleDeposit(
				eh.domainID, d.DestinationDomainID, d.DepositNonce, d.ResourceID, d.Data, d.HandlerResponse,
			)
			if err != nil {
				_ = observability.LogAndRecordError(&logger, span, err, "failed to HandleDeposit", append(d.TraceEventAttributes(), attribute.Int("deposit.srcdomainId", int(eh.domainID)))...)
				return
			}
			observability.LogAndEvent(logger.Info(), span, "Resolved deposit message", attribute.String("msg.id", m.ID()), attribute.String("msg.full", m.String()))
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
	return &RetryEventHandler{
		log:                logC.Logger(),
		eventListener:      eventListener,
		depositHandler:     depositHandler,
		bridgeAddress:      bridgeAddress,
		bridgeABI:          bridgeABI,
		domainID:           domainID,
		blockConfirmations: blockConfirmations,
	}
}

func (eh *RetryEventHandler) HandleEvent(
	ctx context.Context,
	startBlock *big.Int,
	endBlock *big.Int,
	msgChan chan []*message.Message,
) error {
	ctx, span, logger := observability.CreateSpanAndLoggerFromContext(ctx, "relayer-sygma", "relayer.sygma.RetryEventHandler.HandleEvent", attribute.String("startBlock", startBlock.String()), attribute.String("endBlock", endBlock.String()))
	defer span.End()
	retryEvents, err := eh.eventListener.FetchRetryEvents(ctx, eh.bridgeAddress, startBlock, endBlock)
	if err != nil {
		return observability.LogAndRecordErrorWithStatus(&logger, span, err, "unable to fetch retry events because")
	}
	retriesByDomain := make(map[uint8][]*message.Message)
	for _, event := range retryEvents {
		func(event hubEvents.RetryEvent) {
			defer func() {
				if r := recover(); r != nil {
					_ = observability.LogAndRecordError(&logger, span, fmt.Errorf("panic"), "failed to handle retry event")
				}
			}()

			deposits, err := eh.eventListener.FetchDepositEvent(event, eh.bridgeAddress, eh.blockConfirmations)
			if err != nil {
				_ = observability.LogAndRecordError(&logger, span, err, "failed to fetch DepositEvent for retry")
				return
			}

			for _, d := range deposits {
				msg, err := eh.depositHandler.HandleDeposit(
					eh.domainID, d.DestinationDomainID, d.DepositNonce,
					d.ResourceID, d.Data, d.HandlerResponse,
				)
				if err != nil {
					_ = observability.LogAndRecordError(&logger, span, err, "failed to fetch HandleDeposit for retry")
					continue
				}
				observability.LogAndEvent(logger.Info(), span, "Resolved retry message", attribute.String("event.retry.hashToRetry", event.TxHash), attribute.String("msg.id", msg.ID()), attribute.String("msg.type", string(msg.Type)))
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
	ctx context.Context,
	startBlock *big.Int,
	endBlock *big.Int,
	msgChan chan []*message.Message,
) error {
	ctx, span, logger := observability.CreateSpanAndLoggerFromContext(ctx, "relayer-sygma", "relayer.sygma.KeygenEventHandler.HandleEvent", attribute.String("startBlock", startBlock.String()), attribute.String("endBlock", endBlock.String()))
	defer span.End()
	key, err := eh.storer.GetKeyshare()
	if (key.Threshold != 0) && (err == nil) {
		span.SetStatus(codes.Ok, "Keyshare already generated")
		return nil
	}

	keygenEvents, err := eh.eventListener.FetchKeygenEvents(
		ctx, eh.bridgeAddress, startBlock, endBlock,
	)
	if err != nil {
		return observability.LogAndRecordErrorWithStatus(&logger, span, err, "unable to fetch keygen events")
	}

	if len(keygenEvents) == 0 {
		span.SetStatus(codes.Ok, "No Keygen events to handle")
		return nil
	}
	observability.LogAndEvent(logger.Info(), span, "Resolved keygen message", attribute.String("event.keygen.tx.hash", keygenEvents[0].TxHash.String()))
	keygenBlockNumber := big.NewInt(0).SetUint64(keygenEvents[0].BlockNumber)
	kg := keygen.NewKeygen(
		eh.sessionID(keygenBlockNumber),
		eh.threshold,
		eh.host,
		eh.communication,
		eh.storer)
	return eh.coordinator.Execute(ctx, kg, make(chan interface{}, 1))
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
	ctx context.Context,
	startBlock *big.Int,
	endBlock *big.Int,
	msgChan chan []*message.Message,
) error {
	ctx, span, logger := observability.CreateSpanAndLoggerFromContext(ctx, "relayer-sygma", "relayer.sygma.RefreshEventHandler.HandleEvent", attribute.String("startBlock", startBlock.String()), attribute.String("endBlock", endBlock.String()))
	defer span.End()
	refreshEvents, err := eh.eventListener.FetchRefreshEvents(
		ctx, eh.bridgeAddress, startBlock, endBlock,
	)
	if err != nil {
		return observability.LogAndRecordErrorWithStatus(&logger, span, err, "unable to fetch keygen events")
	}
	if len(refreshEvents) == 0 {
		return nil
	}

	hash := refreshEvents[len(refreshEvents)-1].Hash
	if hash == "" {
		return observability.LogAndRecordErrorWithStatus(&logger, span, fmt.Errorf("hash cannot be empty string"), "unable to handle refresh event")
	}
	observability.SetSpanAndLoggerAttrs(&logger, span, attribute.String("topology.hash", hash))
	topology, err := eh.topologyProvider.NetworkTopology(hash)
	if err != nil {
		return observability.LogAndRecordErrorWithStatus(&logger, span, err, "unable to fetch topology")

	}
	err = eh.topologyStore.StoreTopology(topology)
	if err != nil {
		return observability.LogAndRecordErrorWithStatus(&logger, span, err, "unable to store topology")
	}

	eh.connectionGate.SetTopology(topology)
	p2p.LoadPeers(eh.host, topology.Peers)

	observability.LogAndEvent(logger.Info(), span, "Resolved refresh message", attribute.String("topology.map", topology.String()))

	resharing := resharing.NewResharing(
		eh.sessionID(startBlock),
		topology.Threshold,
		eh.host,
		eh.communication,
		eh.storer,
	)
	span.SetStatus(codes.Ok, "Resharing event handled")
	return eh.coordinator.Execute(ctx, resharing, make(chan interface{}, 1))
}

func (eh *RefreshEventHandler) sessionID(block *big.Int) string {
	return fmt.Sprintf("resharing-%s", block.String())
}
