package tss

import (
	"context"
	"time"

	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
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
	communication communication.Communication
}

func NewCoordinator(
	host host.Host,
	tssProcess TssProcess,
	communication communication.Communication,
) *Coordinator {
	return &Coordinator{
		host:          host,
		communication: communication,
	}
}

// Execute calculates process leader and coordinates party readiness and start the tss processes.
func (c *Coordinator) Execute(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}, statusChn chan error) {
	sessionID := tssProcess.SessionID()
	errChn := make(chan error)
	if c.isLeader(sessionID) {
		go c.initiate(ctx, tssProcess, resultChn, errChn)
	} else {
		go c.waitForStart(ctx, tssProcess, resultChn, errChn)
	}

	err := <-errChn
	tssProcess.Stop()
	if err != nil {
		log.Err(err)
		statusChn <- err
		return
	}

	statusChn <- nil
}

// IsLeader returns if the peer is the leader for the current
// tss process.
func (c *Coordinator) isLeader(sessionID string) bool {
	peers := c.host.Peerstore().Peers()
	return c.host.ID().Pretty() == common.SortPeersForSession(peers, sessionID)[0].ID.Pretty()
}

// broadcastInitiateMsg sends TssInitiateMsg to all peers
func (c *Coordinator) broadcastInitiateMsg(sessionID string) {
	log.Debug().Msgf("broadcasted initiate message for session: %s")
	go c.communication.Broadcast(
		c.host.Peerstore().Peers(), []byte{}, communication.TssInitiateMsg, sessionID, nil,
	)
}

// initiate sends initiate message to all peers and waits
// for ready response. After tss process declares that enough
// peers are ready, start message is broadcasted and tss process is started.
func (c *Coordinator) initiate(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}, errChn chan error) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	readyChan := make(chan *communication.WrappedMessage)
	readyMap := make(map[peer.ID]bool)
	readyMap[c.host.ID()] = true

	subID := c.communication.Subscribe(tssProcess.SessionID(), communication.TssReadyMsg, readyChan)
	defer c.communication.UnSubscribe(subID)

	c.broadcastInitiateMsg(tssProcess.SessionID())
	for {
		select {
		case wMsg := <-readyChan:
			{
				log.Debug().Msgf("received ready message from %s for session %s", wMsg.From, tssProcess.SessionID())
				readyMap[wMsg.From] = true
				if !tssProcess.Ready(readyMap) {
					continue
				}

				startParams := tssProcess.StartParams(readyMap)
				startMsgBytes, err := common.MarshalStartMessage(startParams)
				if err != nil {
					errChn <- err
					return
				}

				go c.communication.Broadcast(c.host.Peerstore().Peers(), startMsgBytes, communication.TssStartMsg, tssProcess.SessionID(), nil)
				go tssProcess.Start(ctx, resultChn, errChn, startParams)
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

	for {
		select {
		case wMsg := <-msgChan:
			{
				log.Debug().Msgf("sent ready message to %s for session %s", wMsg.From, tssProcess.SessionID())
				go c.communication.Broadcast(
					peer.IDSlice{wMsg.From}, []byte{}, communication.TssReadyMsg, tssProcess.SessionID(), nil,
				)
			}
		case startMsg := <-startMsgChn:
			{
				log.Debug().Msgf("received start message from %s for session %s", startMsg.From, tssProcess.SessionID())

				msg, err := common.UnmarshalStartMessage(startMsg.Payload)
				if err != nil {
					errChn <- err
					return
				}

				go tssProcess.Start(ctx, resultChn, errChn, msg.Params)
				return
			}
		case <-ctx.Done():
			{
				return
			}
		}
	}
}
