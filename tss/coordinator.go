package tss

import (
	"context"
	"time"

	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type TssProcess interface {
	Start(ctx context.Context, resultChn chan interface{}, errChn chan error, params []string)
	Stop()
	Ready(readyMap map[peer.ID]bool) bool
	StartParams(readyMap map[peer.ID]bool) []string
	SessionID() string
}

type Coordinator struct {
	host          host.Host
	tssProcess    TssProcess
	communication communication.Communication
	log           zerolog.Logger
}

func NewCoordinator(
	host host.Host,
	tssProcess TssProcess,
	communication communication.Communication,
) *Coordinator {
	return &Coordinator{
		host:          host,
		tssProcess:    tssProcess,
		communication: communication,
		log:           log.With().Str("SessionID", string(tssProcess.SessionID())).Logger(),
	}
}

// Execute calculates process leader and coordinates party readiness and start the tss processes.
func (c *Coordinator) Execute(ctx context.Context, resultChn chan interface{}, statusChn chan error) {
	errChn := make(chan error)
	if c.isLeader() {
		go c.initiate(ctx, resultChn, errChn)
	} else {
		go c.waitForStart(ctx, resultChn, errChn)
	}

	err := <-errChn
	c.tssProcess.Stop()
	if err != nil {
		log.Err(err)
		statusChn <- err
		return
	}

	statusChn <- nil
}

// IsLeader returns if the peer is the leader for the current
// tss process.
func (c *Coordinator) isLeader() bool {
	peers := c.host.Peerstore().Peers()
	sessionID := c.tssProcess.SessionID()
	return c.host.ID().Pretty() == common.SortPeersForSession(peers, sessionID)[0].ID.Pretty()
}

// broadcastInitiateMsg sends TssInitiateMsg to all peers
func (c *Coordinator) broadcastInitiateMsg() {
	c.log.Debug().Msgf("broadcasted initiate message")
	go c.communication.Broadcast(
		c.host.Peerstore().Peers(), []byte{}, communication.TssInitiateMsg, c.tssProcess.SessionID(), nil,
	)
}

// initiate sends initiate message to all peers and waits
// for ready response. After tss process declares that enough
// peers are ready, start message is broadcasted and tss process is started.
func (c *Coordinator) initiate(ctx context.Context, resultChn chan interface{}, errChn chan error) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	readyChan := make(chan *communication.WrappedMessage)
	readyMap := make(map[peer.ID]bool)
	readyMap[c.host.ID()] = true

	subID := c.communication.Subscribe(c.tssProcess.SessionID(), communication.TssReadyMsg, readyChan)
	defer c.communication.UnSubscribe(subID)

	c.broadcastInitiateMsg()

	for {
		select {
		case wMsg := <-readyChan:
			{
				c.log.Debug().Msgf("received ready message from %s", wMsg.From)
				readyMap[wMsg.From] = true
				if !c.tssProcess.Ready(readyMap) {
					continue
				}

				startParams := c.tssProcess.StartParams(readyMap)
				startMsgBytes, err := common.MarshalStartMessage(startParams)
				if err != nil {
					errChn <- err
					return
				}

				go c.communication.Broadcast(c.host.Peerstore().Peers(), startMsgBytes, communication.TssStartMsg, c.tssProcess.SessionID(), nil)
				go c.tssProcess.Start(ctx, resultChn, errChn, startParams)
				return
			}
		case <-ticker.C:
			{
				c.broadcastInitiateMsg()
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
func (c *Coordinator) waitForStart(ctx context.Context, resultChn chan interface{}, errChn chan error) {
	msgChan := make(chan *communication.WrappedMessage)
	startMsgChn := make(chan *communication.WrappedMessage)

	initSubID := c.communication.Subscribe(c.tssProcess.SessionID(), communication.TssInitiateMsg, msgChan)
	defer c.communication.UnSubscribe(initSubID)
	startSubID := c.communication.Subscribe(c.tssProcess.SessionID(), communication.TssStartMsg, startMsgChn)
	defer c.communication.UnSubscribe(startSubID)

	for {
		select {
		case wMsg := <-msgChan:
			{
				c.log.Debug().Msgf("sent ready message to %s", wMsg.From)
				go c.communication.Broadcast(
					peer.IDSlice{wMsg.From}, []byte{}, communication.TssReadyMsg, c.tssProcess.SessionID(), nil,
				)
			}
		case startMsg := <-startMsgChn:
			{
				c.log.Debug().Msgf("received start message from %s", startMsg.From)

				msg, err := common.UnmarshalStartMessage(startMsg.Payload)
				if err != nil {
					errChn <- err
					return
				}

				go c.tssProcess.Start(ctx, resultChn, errChn, msg.Params)
				return
			}
		case <-ctx.Done():
			{
				return
			}
		}
	}
}
