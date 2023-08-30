// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package tss

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	traceapi "go.opentelemetry.io/otel/trace"
	"sync"
	"time"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/comm/elector"
	"github.com/ChainSafe/sygma-relayer/tss/common"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sourcegraph/conc/pool"
	"golang.org/x/exp/slices"
)

var (
	initiatePeriod     = 15 * time.Second
	coordinatorTimeout = 3 * time.Minute
	tssTimeout         = 15 * time.Minute
)

type TssProcess interface {
	Run(ctx context.Context, coordinator bool, resultChn chan interface{}, params []byte) error
	Stop()
	Ready(readyMap map[peer.ID]bool, excludedPeers []peer.ID) (bool, error)
	Retryable() bool
	StartParams(readyMap map[peer.ID]bool) []byte
	SessionID() string
	ValidCoordinators() []peer.ID
}

type Coordinator struct {
	host           host.Host
	communication  comm.Communication
	electorFactory *elector.CoordinatorElectorFactory

	pendingProcesses map[string]bool
	processLock      sync.Mutex

	CoordinatorTimeout time.Duration
	TssTimeout         time.Duration
	InitiatePeriod     time.Duration
}

func NewCoordinator(
	host host.Host,
	communication comm.Communication,
	electorFactory *elector.CoordinatorElectorFactory,
) *Coordinator {
	return &Coordinator{
		host:           host,
		communication:  communication,
		electorFactory: electorFactory,

		pendingProcesses: make(map[string]bool),

		CoordinatorTimeout: coordinatorTimeout,
		TssTimeout:         tssTimeout,
		InitiatePeriod:     initiatePeriod,
	}
}

// Execute calculates process leader and coordinates party readiness and start the tss processes.
func (c *Coordinator) Execute(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}) error {
	ctx, span := otel.Tracer("relayer-sygma").Start(ctx, "relayer.sygma.Coordinator.Execute")
	defer span.End()
	logger := log.With().Str("dd.trace_id", span.SpanContext().TraceID().String()).Logger()
	sessionID := tssProcess.SessionID()
	value, ok := c.pendingProcesses[sessionID]
	if ok && value {
		logger.Warn().Str("SessionID", sessionID).Msgf("Process already pending")
		span.SetStatus(codes.Error, "process already pending")
		return fmt.Errorf("process already pending")
	}

	c.processLock.Lock()
	c.pendingProcesses[sessionID] = true
	c.processLock.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	p := pool.New().WithContext(ctx).WithCancelOnError()
	defer func() {
		cancel()
		c.communication.CloseSession(sessionID)
		c.processLock.Lock()
		c.pendingProcesses[sessionID] = false
		c.processLock.Unlock()
	}()

	coordinatorElector := c.electorFactory.CoordinatorElector(sessionID, elector.Static)
	coordinator, _ := coordinatorElector.Coordinator(ctx, tssProcess.ValidCoordinators())

	logger.Info().Str("SessionID", sessionID).Msgf("Starting process with coordinator %s", coordinator.String())
	span.AddEvent("Coordinator selected", traceapi.WithAttributes(attribute.String("tss.coordinator", coordinator.String())))
	p.Go(func(ctx context.Context) error {
		err := c.start(ctx, tssProcess, coordinator, resultChn, []peer.ID{})
		if err == nil {
			cancel()
		}
		return err
	})
	p.Go(func(ctx context.Context) error {
		return c.watchExecution(ctx, tssProcess, coordinator)
	})
	err := p.Wait()
	if err == nil {
		return nil
	}

	span.RecordError(err)
	if !tssProcess.Retryable() {
		span.SetStatus(codes.Error, "Process is not retryable. Returning error")
		return err
	}
	span.AddEvent("Retrying tssProcess")
	return c.handleError(ctx, err, tssProcess, resultChn)
}

func (c *Coordinator) handleError(ctx context.Context, err error, tssProcess TssProcess, resultChn chan interface{}) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rp := pool.New().WithContext(ctx).WithCancelOnError()
	rp.Go(func(ctx context.Context) error {
		return c.watchExecution(ctx, tssProcess, peer.ID(""))
	})
	switch err := err.(type) {
	case *CoordinatorError:
		{
			log.Err(err).Str("SessionID", tssProcess.SessionID()).Msgf("Tss process failed with error %+v", err)

			excludedPeers := []peer.ID{err.Peer}
			rp.Go(func(ctx context.Context) error { return c.retry(ctx, tssProcess, resultChn, excludedPeers) })
		}
	case *comm.CommunicationError:
		{
			log.Err(err).Str("SessionID", tssProcess.SessionID()).Msgf("Tss process failed with error %+v", err)
			rp.Go(func(ctx context.Context) error { return c.retry(ctx, tssProcess, resultChn, []peer.ID{}) })
		}
	case *tss.Error:
		{
			log.Err(err).Str("SessionID", tssProcess.SessionID()).Msgf("Tss process failed with error %+v", err)
			excludedPeers, err := common.PeersFromParties(err.Culprits())
			if err != nil {
				return err
			}
			rp.Go(func(ctx context.Context) error { return c.retry(ctx, tssProcess, resultChn, excludedPeers) })
		}
	case *SubsetError:
		{
			// wait for start message if existing singing process fails
			rp.Go(func(ctx context.Context) error {
				return c.waitForStart(ctx, tssProcess, resultChn, peer.ID(""), c.TssTimeout)
			})
		}
	default:
		{
			return err
		}
	}
	return rp.Wait()
}

func (c *Coordinator) watchExecution(ctx context.Context, tssProcess TssProcess, coordinator peer.ID) error {
	failChn := make(chan *comm.WrappedMessage)
	subscriptionID := c.communication.Subscribe(tssProcess.SessionID(), comm.TssFailMsg, failChn)
	ticker := time.NewTicker(c.TssTimeout)
	defer func() {
		ticker.Stop()
		c.communication.UnSubscribe(subscriptionID)
	}()

	for {
		select {
		case <-ticker.C:
			{
				return fmt.Errorf("tss process timed out after %v", c.TssTimeout)
			}
		case <-ctx.Done():
			{
				return nil
			}
		case msg := <-failChn:
			{
				// ignore messages that are not from coordinator
				if msg.From.String() != coordinator.String() {
					continue
				}

				return fmt.Errorf("tss fail message received for process %s", tssProcess.SessionID())
			}
		}
	}
}

// start initiates listeners for coordinator and participants with static calculated coordinator
func (c *Coordinator) start(ctx context.Context, tssProcess TssProcess, coordinator peer.ID, resultChn chan interface{}, excludedPeers []peer.ID) error {
	if coordinator.String() == c.host.ID().String() {
		return c.initiate(ctx, tssProcess, resultChn, excludedPeers)
	} else {
		return c.waitForStart(ctx, tssProcess, resultChn, coordinator, c.CoordinatorTimeout)
	}
}

// retry initiates full bully process to calculate coordinator and starts a new tss process after
// an expected error ocurred during regular tss execution
func (c *Coordinator) retry(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}, excludedPeers []peer.ID) error {
	coordinatorElector := c.electorFactory.CoordinatorElector(tssProcess.SessionID(), elector.Bully)
	coordinator, err := coordinatorElector.Coordinator(ctx, common.ExcludePeers(tssProcess.ValidCoordinators(), excludedPeers))
	if err != nil {
		return err
	}

	return c.start(ctx, tssProcess, coordinator, resultChn, excludedPeers)
}

// broadcastInitiateMsg sends TssInitiateMsg to all peers
func (c *Coordinator) broadcastInitiateMsg(ctx context.Context, sessionID string) {
	log.Debug().Msgf("broadcasted initiate message for session: %s", sessionID)
	_ = c.communication.Broadcast(
		ctx, c.host.Peerstore().Peers(), []byte{}, comm.TssInitiateMsg, sessionID,
	)
}

// initiate sends initiate message to all peers and waits
// for ready response. After tss process declares that enough
// peers are ready, start message is broadcasted and tss process is started.
func (c *Coordinator) initiate(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}, excludedPeers []peer.ID) error {
	ctx, span := otel.Tracer("relayer-sygma").Start(ctx, "relayer.sygma.tss.Coordinator.initiate")
	defer span.End()
	logger := log.With().Str("dd.trace_id", span.SpanContext().TraceID().String()).Logger()
	readyChan := make(chan *comm.WrappedMessage)
	readyMap := make(map[peer.ID]bool)
	readyMap[c.host.ID()] = true

	subID := c.communication.Subscribe(tssProcess.SessionID(), comm.TssReadyMsg, readyChan)
	defer c.communication.UnSubscribe(subID)

	ticker := time.NewTicker(c.InitiatePeriod)
	defer ticker.Stop()
	c.broadcastInitiateMsg(ctx, tssProcess.SessionID())
	for {
		select {
		case wMsg := <-readyChan:
			{
				logger.Debug().Str("SessionID", tssProcess.SessionID()).Msgf("received ready message from %s", wMsg.From)
				span.AddEvent("Received ready message", traceapi.WithAttributes(attribute.String("tss.msg.from", wMsg.From.String()), attribute.String("tss.msg.sessionID", tssProcess.SessionID())))
				if !slices.Contains(excludedPeers, wMsg.From) {
					readyMap[wMsg.From] = true
				}
				ready, err := tssProcess.Ready(readyMap, excludedPeers)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())
					return err
				}
				if !ready {
					continue
				}
				span.AddEvent("Ready for start", traceapi.WithAttributes(attribute.String("tss.session.id", tssProcess.SessionID())))
				startParams := tssProcess.StartParams(readyMap)
				startMsgBytes, err := common.MarshalStartMessage(startParams)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())
					return err
				}

				_ = c.communication.Broadcast(ctx, c.host.Peerstore().Peers(), startMsgBytes, comm.TssStartMsg, tssProcess.SessionID())
				return tssProcess.Run(ctx, true, resultChn, startParams)
			}
		case <-ticker.C:
			{
				c.broadcastInitiateMsg(ctx, tssProcess.SessionID())
			}
		case <-ctx.Done():
			{
				return nil
			}
		}
	}
}

// waitForStart responds to initiate messages and starts the tss process
// when it receives the start message.
func (c *Coordinator) waitForStart(
	ctx context.Context,
	tssProcess TssProcess,
	resultChn chan interface{},
	coordinator peer.ID,
	timeout time.Duration,
) error {
	ctxWithInternalSpan, internalSpan := otel.Tracer("relayer-sygma").Start(ctx, "relayer.sygma.tss.Coordinator.waitForStart")
	defer internalSpan.End()

	msgChan := make(chan *comm.WrappedMessage)
	startMsgChn := make(chan *comm.WrappedMessage)

	initSubID := c.communication.Subscribe(tssProcess.SessionID(), comm.TssInitiateMsg, msgChan)
	defer c.communication.UnSubscribe(initSubID)
	startSubID := c.communication.Subscribe(tssProcess.SessionID(), comm.TssStartMsg, startMsgChn)
	defer c.communication.UnSubscribe(startSubID)

	coordinatorTimeoutTicker := time.NewTicker(timeout)
	defer coordinatorTimeoutTicker.Stop()
	for {
		select {
		case wMsg := <-msgChan:
			{
				tID, err := traceapi.TraceIDFromHex(wMsg.TraceID)
				if err != nil {
					log.Warn().Str("traceID", wMsg.TraceID).Msg("TraceID is wrong")
				}

				ctxWithSpan, span := otel.Tracer("relayer-sygma").Start(traceapi.ContextWithSpanContext(ctxWithInternalSpan, traceapi.NewSpanContext(traceapi.SpanContextConfig{TraceID: tID, Remote: true})), "relayer.sygma.tss.Coordinator.InitMessage")
				logger := log.With().Str("dd.trace_id", span.SpanContext().TraceID().String()).Logger()

				coordinatorTimeoutTicker.Reset(timeout)
				span.AddEvent("Received initiate message", traceapi.WithAttributes(attribute.String("tss.msg.coordinator", wMsg.From.String()), attribute.String("tss.session.id", tssProcess.SessionID())))
				logger.Debug().Str("SessionID", tssProcess.SessionID()).Msgf("sent ready message to %s", wMsg.From)
				_ = c.communication.Broadcast(
					ctxWithSpan, peer.IDSlice{wMsg.From}, []byte{}, comm.TssReadyMsg, tssProcess.SessionID(),
				)
				span.End()
			}
		case startMsg := <-startMsgChn:
			{
				tID, err := traceapi.TraceIDFromHex(startMsg.TraceID)
				if err != nil {
					log.Warn().Str("traceID", startMsg.TraceID).Msg("TraceID is wrong")
				}
				ctxWithRemoteSpan, span := otel.Tracer("relayer-sygma").Start(traceapi.ContextWithSpanContext(ctxWithInternalSpan, traceapi.NewSpanContext(traceapi.SpanContextConfig{TraceID: tID, Remote: true})), "relayer.sygma.tss.Coordinator.StartMessage")
				defer span.End()
				logger := log.With().Str("dd.trace_id", span.SpanContext().TraceID().String()).Logger()

				logger.Debug().Str("SessionID", tssProcess.SessionID()).Msgf("received start message from %s", startMsg.From)
				span.AddEvent("Received start message", traceapi.WithAttributes(attribute.String("tss.msg.coordinator", startMsg.From.String()), attribute.String("tss.session.id", tssProcess.SessionID())))

				// having startMsg.From as "" is special case when peer is not selected in subset
				// but should wait for start message if existing singing process fails
				if coordinator != "" && startMsg.From != coordinator {
					err := fmt.Errorf(
						"start message received from peer %s that is not coordinator %s",
						startMsg.From.String(), coordinator.String(),
					)
					span.SetStatus(codes.Error, err.Error())

					return err
				}

				msg, err := common.UnmarshalStartMessage(startMsg.Payload)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())
					return err
				}

				return tssProcess.Run(ctxWithRemoteSpan, false, resultChn, msg.Params)
			}
		case <-coordinatorTimeoutTicker.C:
			{
				return &CoordinatorError{Peer: coordinator}
			}
		case <-ctx.Done():
			{
				return nil
			}
		}
	}
}
