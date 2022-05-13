package tss

import (
	"context"
	"time"

	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

var (
	initiatePeriod     = 15 * time.Second
	coordinatorTimeout = 15 * time.Minute
)

type Bully interface {
	Coordinator(excludedPeers peer.IDSlice, coordinatorChan chan peer.ID, errChan chan error)
}

type TssProcess interface {
	Start(ctx context.Context, coordinator bool, resultChn chan interface{}, errChn chan error, params []string)
	Stop()
	Ready(readyMap map[peer.ID]bool, excludedPeers []peer.ID) (bool, error)
	StartParams(readyMap map[peer.ID]bool) []string
	SessionID() string
}

type Coordinator struct {
	host          host.Host
	communication communication.Communication
	bully         Bully

	CoordinatorTimeout time.Duration
}

func NewCoordinator(
	host host.Host,
	communication communication.Communication,
	bully Bully,
) *Coordinator {
	return &Coordinator{
		host:               host,
		bully:              bully,
		communication:      communication,
		CoordinatorTimeout: coordinatorTimeout,
	}
}

// Execute calculates process leader and coordinates party readiness and start the tss processes.
func (c *Coordinator) Execute(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}, statusChn chan error) {
	errChn := make(chan error)
	coordinator := c.getCoordinator(tssProcess.SessionID())
	go c.start(ctx, tssProcess, coordinator, resultChn, errChn, []peer.ID{})

	retried := false
	defer tssProcess.Stop()
	for {
		select {
		case <-ctx.Done():
			{
				statusChn <- nil
				return
			}
		case err := <-errChn:
			{
				if err == nil {
					statusChn <- nil
					return
				}

				log.Err(err).Msgf("Tss process failed with error: %v", err)

				if retried {
					statusChn <- err
					return
				}

				switch err := err.(type) {
				case *CoordinatorError:
					{
						tssProcess.Stop()
						retried = true
						go c.retry(ctx, tssProcess, resultChn, errChn, []peer.ID{err.Coordinator})
					}
				case *tss.Error:
					{
						tssProcess.Stop()
						retried = true
						excludedPeers, err := common.PeersFromParties(err.Culprits())
						if err != nil {
							statusChn <- err
							return
						}

						go c.retry(ctx, tssProcess, resultChn, errChn, excludedPeers)
					}
				default:
					{
						statusChn <- err
						return
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
		c.waitForStart(ctx, tssProcess, resultChn, errChn)
	}
}

// retry initiates full bully process to calculate coordinator and starts a new tss process after
// an expected error ocurred during regular tss execution
func (c *Coordinator) retry(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}, errChn chan error, excludedPeers []peer.ID) {
	coordinatorChn := make(chan peer.ID)
	c.bully.Coordinator([]peer.ID{c.getCoordinator(tssProcess.SessionID())}, coordinatorChn, errChn)
	coordinator := <-coordinatorChn
	go c.start(ctx, tssProcess, coordinator, resultChn, errChn, excludedPeers)
}

// getLeader returns the static leader for current session
func (c *Coordinator) getCoordinator(sessionID string) peer.ID {
	peers := c.host.Peerstore().Peers()
	return common.SortPeersForSession(peers, sessionID)[0].ID
}

// broadcastInitiateMsg sends TssInitiateMsg to all peers
func (c *Coordinator) broadcastInitiateMsg(sessionID string) {
	log.Debug().Msgf("broadcasted initiate message for session: %s", sessionID)
	go c.communication.Broadcast(
		c.host.Peerstore().Peers(), []byte{}, communication.TssInitiateMsg, sessionID, nil,
	)
}

// initiate sends initiate message to all peers and waits
// for ready response. After tss process declares that enough
// peers are ready, start message is broadcasted and tss process is started.
func (c *Coordinator) initiate(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}, errChn chan error, excludedPeers []peer.ID) {
	readyChan := make(chan *communication.WrappedMessage)
	readyMap := make(map[peer.ID]bool)
	readyMap[c.host.ID()] = true

	subID := c.communication.Subscribe(tssProcess.SessionID(), communication.TssReadyMsg, readyChan)
	defer c.communication.UnSubscribe(subID)

	ticker := time.NewTicker(initiatePeriod)
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

				go c.communication.Broadcast(c.host.Peerstore().Peers(), startMsgBytes, communication.TssStartMsg, tssProcess.SessionID(), nil)
				go tssProcess.Start(ctx, true, resultChn, errChn, startParams)
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
func (c *Coordinator) waitForStart(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}, errChn chan error) {
	msgChan := make(chan *communication.WrappedMessage)
	startMsgChn := make(chan *communication.WrappedMessage)

	initSubID := c.communication.Subscribe(tssProcess.SessionID(), communication.TssInitiateMsg, msgChan)
	defer c.communication.UnSubscribe(initSubID)
	startSubID := c.communication.Subscribe(tssProcess.SessionID(), communication.TssStartMsg, startMsgChn)
	defer c.communication.UnSubscribe(startSubID)

	coordinatorTimeoutTicker := time.NewTicker(c.CoordinatorTimeout)
	defer coordinatorTimeoutTicker.Stop()
	for {
		select {
		case wMsg := <-msgChan:
			{
				coordinatorTimeoutTicker.Reset(coordinatorTimeout)

				log.Debug().Str("SessionID", tssProcess.SessionID()).Msgf("sent ready message to %s", wMsg.From)
				go c.communication.Broadcast(
					peer.IDSlice{wMsg.From}, []byte{}, communication.TssReadyMsg, tssProcess.SessionID(), nil,
				)
			}
		case startMsg := <-startMsgChn:
			{
				log.Debug().Str("SessionID", tssProcess.SessionID()).Msgf("received start message from %s", startMsg.From)
				msg, err := common.UnmarshalStartMessage(startMsg.Payload)
				if err != nil {
					errChn <- err
					return
				}

				go tssProcess.Start(ctx, false, resultChn, errChn, msg.Params)
				return
			}
		case <-coordinatorTimeoutTicker.C:
			{
				errChn <- &CoordinatorError{Coordinator: c.getCoordinator(tssProcess.SessionID())}
				return
			}
		case <-ctx.Done():
			{
				return
			}
		}
	}
}
