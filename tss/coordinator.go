package tss

import (
	"context"
	"fmt"
	"github.com/ChainSafe/chainbridge-core/comm/bully"
	"github.com/ChainSafe/chainbridge-core/comm/static"
	"time"

	"github.com/ChainSafe/chainbridge-core/comm"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
)

type TssProcess interface {
	Start(ctx context.Context, coordinator bool, resultChn chan interface{}, errChn chan error, params []string)
	Stop()
	Ready(readyMap map[peer.ID]bool) bool
	StartParams(readyMap map[peer.ID]bool) []string
	SessionID() string
}

type Coordinator struct {
	host          host.Host
	communication comm.Communication
	static        static.CommunicationCoordinator
	bully         bully.CommunicationCoordinatorFactory
}

func NewCoordinator(
	host host.Host,
	communication comm.Communication,
) *Coordinator {
	return &Coordinator{
		host:          host,
		communication: communication,
		static:        static.NewStaticCommunicationCoordinator(host),
	}
}

// Execute calculates process leader and coordinates party readiness and start the tss processes.
func (c *Coordinator) Execute(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}, statusChn chan error) {
	sessionID := tssProcess.SessionID()
	bullyCoordinator := c.bully.NewCommunicationCoordinator(sessionID)
	errChn := make(chan error)
	defer tssProcess.Stop()
	coordinator, _ := c.static.GetCoordinator(sessionID)
	if c.host.ID() == coordinator {
		go c.initiate(ctx, tssProcess, resultChn, errChn)
	} else {
		go c.waitForStart(ctx, tssProcess, resultChn, errChn)
	}

	err := <-errChn
	if err != nil {
		log.Err(err).Msgf("Error occurred during tss process")
		statusChn <- err

		oldCoordinator := coordinator
		newCoordinator, err := bullyCoordinator.GetCoordinator(peer.IDSlice{oldCoordinator})
		if err != nil {
			return
		}
		fmt.Println(newCoordinator)
		return
	}

	statusChn <- nil
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
func (c *Coordinator) initiate(ctx context.Context, tssProcess TssProcess, resultChn chan interface{}, errChn chan error) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	readyChan := make(chan *comm.WrappedMessage)
	readyMap := make(map[peer.ID]bool)
	readyMap[c.host.ID()] = true

	subID := c.communication.Subscribe(tssProcess.SessionID(), comm.TssReadyMsg, readyChan)
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

				go c.communication.Broadcast(c.host.Peerstore().Peers(), startMsgBytes, comm.TssStartMsg, tssProcess.SessionID(), nil)
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
	msgChan := make(chan *comm.WrappedMessage)
	startMsgChn := make(chan *comm.WrappedMessage)

	initSubID := c.communication.Subscribe(tssProcess.SessionID(), comm.TssInitiateMsg, msgChan)
	defer c.communication.UnSubscribe(initSubID)
	startSubID := c.communication.Subscribe(tssProcess.SessionID(), comm.TssStartMsg, startMsgChn)
	defer c.communication.UnSubscribe(startSubID)

	for {
		select {
		case wMsg := <-msgChan:
			{
				log.Debug().Msgf("sent ready message to %s for session %s", wMsg.From, tssProcess.SessionID())
				go c.communication.Broadcast(
					peer.IDSlice{wMsg.From}, []byte{}, comm.TssReadyMsg, tssProcess.SessionID(), nil,
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

				go tssProcess.Start(ctx, false, resultChn, errChn, msg.Params)
				return
			}
		case <-ctx.Done():
			{
				return
			}
		}
	}
}
