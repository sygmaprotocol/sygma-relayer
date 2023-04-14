// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package tss

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/comm/elector"
	"github.com/ChainSafe/sygma-relayer/tss/common"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sourcegraph/conc"
	"golang.org/x/exp/slices"
)

var (
	initiatePeriod     = 15 * time.Second
	coordinatorTimeout = 5 * time.Minute
	tssTimeout         = 30 * time.Minute
)

type TssProcess interface {
	Start(ctx context.Context, coordinator bool, resultChn chan interface{}, errChn chan error, params []byte)
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
	retriedProcesses map[string]bool
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
		retriedProcesses: make(map[string]bool),

		CoordinatorTimeout: coordinatorTimeout,
		TssTimeout:         tssTimeout,
		InitiatePeriod:     initiatePeriod,
	}
}

// Execute calculates process leader and coordinates party readiness and start the tss processes.
func (c *Coordinator) Execute(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}) error {
	sessionID := tssProcess.SessionID()
	value, ok := c.pendingProcesses[sessionID]
	if ok && value {
		log.Warn().Str("SessionID", sessionID).Msgf("Process already pending")
		return fmt.Errorf("process already pending")
	}

	c.processLock.Lock()
	c.pendingProcesses[sessionID] = true
	c.processLock.Unlock()

	failChn := make(chan *comm.WrappedMessage)
	subscriptionID := c.communication.Subscribe(tssProcess.SessionID(), comm.TssFailMsg, failChn)
	ticker := time.NewTicker(c.TssTimeout)

	var wg conc.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		wg.Wait()
		ticker.Stop()
		c.communication.CloseSession(sessionID)
		c.communication.UnSubscribe(subscriptionID)
		c.processLock.Lock()
		c.pendingProcesses[sessionID] = false
		c.retriedProcesses[sessionID] = false
		c.processLock.Unlock()
	}()

	coordinatorElector := c.electorFactory.CoordinatorElector(sessionID, elector.Static)
	coordinator, _ := coordinatorElector.Coordinator(ctx, tssProcess.ValidCoordinators())

	log.Info().Str("SessionID", sessionID).Msgf("Starting process with coordinator %s", coordinator.Pretty())

	errChn := make(chan error)
	wg.Go(func() {
		c.start(ctx, tssProcess, coordinator, resultChn, errChn, []peer.ID{})
	})
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
				if msg.From.Pretty() != coordinator.Pretty() {
					continue
				}

				return fmt.Errorf("tss fail message received for process %s", sessionID)
			}
		case err := <-errChn:
			{
				if err == nil {
					return nil
				}

				if !tssProcess.Retryable() {
					return fmt.Errorf("process failed with error: %+v", err)
				}

				retryError := c.lockRetry(sessionID)
				if retryError != nil {
					// retry is already pending
					continue
				}

				switch err := err.(type) {
				case *CoordinatorError:
					{
						log.Err(err).Str("SessionID", sessionID).Msgf("Tss process failed with error %+v", err)

						excludedPeers := []peer.ID{err.Peer}
						wg.Go(func() { c.retry(ctx, tssProcess, resultChn, errChn, excludedPeers) })
					}
				case *comm.CommunicationError:
					{
						log.Err(err).Str("SessionID", sessionID).Msgf("Tss process failed with error %+v", err)
						wg.Go(func() { c.retry(ctx, tssProcess, resultChn, errChn, []peer.ID{}) })
					}
				case *tss.Error:
					{
						log.Err(err).Str("SessionID", sessionID).Msgf("Tss process failed with error %+v", err)
						excludedPeers, err := common.PeersFromParties(err.Culprits())
						if err != nil {
							return err
						}
						wg.Go(func() { c.retry(ctx, tssProcess, resultChn, errChn, excludedPeers) })
					}
				case *SubsetError:
					{
						// wait for start message if existing singing process fails
						wg.Go(func() { c.waitForStart(ctx, tssProcess, resultChn, errChn, peer.ID(""), c.TssTimeout) })
					}
				default:
					{
						return err
					}
				}
			}
		}
	}
}

// start initiates listeners for coordinator and participants with static calculated coordinator
func (c *Coordinator) start(ctx context.Context, tssProcess TssProcess, coordinator peer.ID, resultChn chan interface{}, errChn chan error, excludedPeers []peer.ID) {
	if coordinator.Pretty() == c.host.ID().Pretty() {
		c.initiate(ctx, tssProcess, resultChn, errChn, excludedPeers)
	} else {
		c.waitForStart(ctx, tssProcess, resultChn, errChn, coordinator, c.CoordinatorTimeout)
	}
}

// lockRetry checks if a retry already happened and prevents multiple retries happening
// at the same time
func (c *Coordinator) lockRetry(sessionID string) error {
	c.processLock.Lock()
	defer c.processLock.Unlock()

	retried := c.retriedProcesses[sessionID]
	if retried {
		err := fmt.Errorf("process %s has pending retry", sessionID)
		log.Err(err).Msg("retry already locked")
		return err
	}

	c.retriedProcesses[sessionID] = true
	return nil
}

// retry initiates full bully process to calculate coordinator and starts a new tss process after
// an expected error ocurred during regular tss execution
func (c *Coordinator) retry(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}, errChn chan error, excludedPeers []peer.ID) {
	coordinatorElector := c.electorFactory.CoordinatorElector(tssProcess.SessionID(), elector.Bully)
	coordinator, err := coordinatorElector.Coordinator(ctx, common.ExcludePeers(tssProcess.ValidCoordinators(), excludedPeers))
	if err != nil {
		errChn <- err
		return
	}

	c.start(ctx, tssProcess, coordinator, resultChn, errChn, excludedPeers)
}

// broadcastInitiateMsg sends TssInitiateMsg to all peers
func (c *Coordinator) broadcastInitiateMsg(sessionID string) {
	log.Debug().Msgf("broadcasted initiate message for session: %s", sessionID)
	go c.communication.Broadcast(
		c.host.Peerstore().Peers(), []byte{}, comm.TssInitiateMsg, sessionID, nil,
	)
}

// initiate sends initiate message to all peers and waits
// for ready response. After tss process declares that enough
// peers are ready, start message is broadcasted and tss process is started.
func (c *Coordinator) initiate(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}, errChn chan error, excludedPeers []peer.ID) {
	readyChan := make(chan *comm.WrappedMessage)
	readyMap := make(map[peer.ID]bool)
	readyMap[c.host.ID()] = true

	subID := c.communication.Subscribe(tssProcess.SessionID(), comm.TssReadyMsg, readyChan)
	defer c.communication.UnSubscribe(subID)

	ticker := time.NewTicker(c.InitiatePeriod)
	defer ticker.Stop()
	c.broadcastInitiateMsg(tssProcess.SessionID())
	for {
		select {
		case wMsg := <-readyChan:
			{
				log.Debug().Str("SessionID", tssProcess.SessionID()).Msgf("received ready message from %s", wMsg.From)
				if !slices.Contains(excludedPeers, wMsg.From) {
					readyMap[wMsg.From] = true
				}
				ready, err := tssProcess.Ready(readyMap, excludedPeers)
				if err != nil {
					errChn <- err
					return
				}
				if !ready {
					continue
				}

				startParams := tssProcess.StartParams(readyMap)
				startMsgBytes, err := common.MarshalStartMessage(startParams)
				if err != nil {
					errChn <- err
					return
				}

				c.communication.Broadcast(c.host.Peerstore().Peers(), startMsgBytes, comm.TssStartMsg, tssProcess.SessionID(), nil)
				tssProcess.Start(ctx, true, resultChn, errChn, startParams)
				return
			}
		case <-ticker.C:
			{
				c.broadcastInitiateMsg(tssProcess.SessionID())
			}
		case <-ctx.Done():
			{
				return
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
	errChn chan error,
	coordinator peer.ID,
	timeout time.Duration,
) {
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
				coordinatorTimeoutTicker.Reset(timeout)

				log.Debug().Str("SessionID", tssProcess.SessionID()).Msgf("sent ready message to %s", wMsg.From)
				go c.communication.Broadcast(
					peer.IDSlice{wMsg.From}, []byte{}, comm.TssReadyMsg, tssProcess.SessionID(), nil,
				)
			}
		case startMsg := <-startMsgChn:
			{
				log.Debug().Str("SessionID", tssProcess.SessionID()).Msgf("received start message from %s", startMsg.From)

				// having startMsg.From as "" is special case when peer is not selected in subset
				// but should wait for start message if existing singing process fails
				if coordinator != "" && startMsg.From != coordinator {
					errChn <- fmt.Errorf(
						"start message received from peer %s that is not coordinator %s",
						startMsg.From.Pretty(), coordinator.Pretty(),
					)
					break
				}

				msg, err := common.UnmarshalStartMessage(startMsg.Payload)
				if err != nil {
					errChn <- err
					return
				}

				tssProcess.Start(ctx, false, resultChn, errChn, msg.Params)
				return
			}
		case <-coordinatorTimeoutTicker.C:
			{
				errChn <- &CoordinatorError{Peer: coordinator}
				return
			}
		case <-ctx.Done():
			{
				return
			}
		}
	}
}
