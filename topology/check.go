package topology

import (
	"github.com/ChainSafe/sygma/comm"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

type CommLivelinessChecker struct {
	communication comm.Communication
	peers         peer.IDSlice
	hostID        peer.ID
}

const sessionID = "LivelinessSession"

func NewCommLivelinessChecker(communication comm.Communication, allPeers peer.IDSlice, hostID peer.ID) CommLivelinessChecker {
	return CommLivelinessChecker{
		communication: communication,
		peers:         allPeers,
		hostID:        hostID,
	}
}

func (c *CommLivelinessChecker) StartCheck() {
	inboundPingMsgChannel := make(chan *comm.WrappedMessage)
	subIDPing := c.communication.Subscribe(sessionID, comm.CoordinatorPingMsg, inboundPingMsgChannel)

	go func() {
		for {
			select {
			case msg := <-inboundPingMsgChannel:
				c.communication.Broadcast(
					[]peer.ID{msg.From}, []byte{}, comm.CoordinatorPingResponseMsg, sessionID, nil,
				)
			}
		}
	}()

	inboundPingResponseMsgChannel := make(chan *comm.WrappedMessage)
	subIDPingResp := c.communication.Subscribe(sessionID, comm.CoordinatorPingResponseMsg, inboundPingResponseMsgChannel)

	responsivePeers := map[peer.ID]bool{}
	mux := &sync.RWMutex{}

	go func() {
		for {
			select {
			case msg := <-inboundPingResponseMsgChannel:
				mux.Lock()
				log.Info().Msgf("Peer %s is responsive", msg.From.Pretty())
				responsivePeers[msg.From] = true
				mux.Unlock()
			}
		}
	}()

	for {
		var notResp peer.IDSlice
		mux.Lock()
		for _, p := range c.peers {
			if !responsivePeers[p] && p != c.hostID {
				notResp = append(notResp, p)
			}
		}
		mux.Unlock()

		if len(notResp) == 0 {
			break
		}

		log.Info().Msgf("Not responsive relayers are: %+v", notResp)

		c.communication.Broadcast(notResp, []byte{}, comm.CoordinatorPingMsg, sessionID, nil)

		time.Sleep(5 * time.Second)
	}

	log.Info().Msg("Connections checked")
	c.communication.UnSubscribe(subIDPing)
	c.communication.UnSubscribe(subIDPingResp)
}
