package tss

import (
	"context"
	"time"

	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

type TssProcess interface {
	Start(ctx context.Context, params []string)
	Stop()
	Ready(readyMap map[peer.ID]bool) bool
	SessionID() string
	StartParams() []string
}

type Coordinator struct {
	host          host.Host
	tssProcess    TssProcess
	communication common.Communication
	errChn        chan error
}

func NewCoordinator(
	host host.Host,
	tssProcess TssProcess,
	communication common.Communication,
	errChn chan error,
) *Coordinator {
	return &Coordinator{
		host:          host,
		tssProcess:    tssProcess,
		communication: communication,
		errChn:        errChn,
	}
}

// Execute calculates process leader and coordinates party readiness.
func (c *Coordinator) Execute(ctx context.Context) {
	if c.IsLeader() {
		go c.initiate(ctx)
	} else {
		go c.waitForStart(ctx)
	}
}

// initiate sends initiate message to all peers and waits
// for ready response. After tss process declares that enough
// peers are ready, start message is broadcasted and tss process is started.
func (c *Coordinator) initiate(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	readyChan := make(chan *common.WrappedMessage)
	readyMap := make(map[peer.ID]bool)
	readyMap[c.host.ID()] = true

	c.communication.Subscribe(common.ReadyMsg, c.tssProcess.SessionID(), readyChan)
	defer c.communication.CancelSubscribe(common.ReadyMsg, c.tssProcess.SessionID())

	go c.communication.Broadcast(c.host.Peerstore().Peers(), []byte{}, common.InitiateMsg, c.tssProcess.SessionID())
	for {
		select {
		case wMsg := <-readyChan:
			{
				readyMap[wMsg.From] = true
				if !c.tssProcess.Ready(readyMap) {
					continue
				}

				startParams := c.tssProcess.StartParams()
				startMsgBytes, err := common.MarshalStartMessage(startParams)
				if err != nil {
					c.errChn <- err
					return
				}

				go c.communication.Broadcast(c.host.Peerstore().Peers(), startMsgBytes, common.StartMsg, c.tssProcess.SessionID())
				go c.tssProcess.Start(ctx, startParams)
				return
			}
		case <-ticker.C:
			{
				go c.communication.Broadcast(c.host.Peerstore().Peers(), []byte{}, common.InitiateMsg, c.tssProcess.SessionID())
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
func (c *Coordinator) waitForStart(ctx context.Context) {
	msgChan := make(chan *common.WrappedMessage)
	startMsgChn := make(chan *common.WrappedMessage)

	c.communication.Subscribe(common.InitiateMsg, c.tssProcess.SessionID(), msgChan)
	defer c.communication.CancelSubscribe(common.InitiateMsg, c.tssProcess.SessionID())
	c.communication.Subscribe(common.StartMsg, c.tssProcess.SessionID(), startMsgChn)
	defer c.communication.CancelSubscribe(common.StartMsg, c.tssProcess.SessionID())

	for {
		select {
		case wMsg := <-msgChan:
			{
				go c.communication.Broadcast(peer.IDSlice{wMsg.From}, []byte{}, common.ReadyMsg, c.tssProcess.SessionID())
			}
		case startMsg := <-startMsgChn:
			{
				msg, err := common.UnmarshalStartMessage(startMsg.Payload)
				if err != nil {
					c.errChn <- err
					return
				}

				go c.tssProcess.Start(ctx, msg.Params)
				return
			}
		case <-ctx.Done():
			{
				return
			}
		}
	}
}

// IsLeader returns if the peer is the leader for the current
// tss process.
func (c *Coordinator) IsLeader() bool {
	peers := c.host.Peerstore().Peers()
	sessionID := c.tssProcess.SessionID()
	return c.host.ID() == common.SortPeersForSession(peers, sessionID)[0].ID
}
