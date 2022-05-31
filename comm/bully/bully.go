package bully

import (
	"github.com/ChainSafe/chainbridge-core/comm"
	"github.com/ChainSafe/chainbridge-core/comm/p2p"
	"github.com/ChainSafe/chainbridge-core/config/relayer"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"sync"
	"time"
)

const ProtocolID protocol.ID = "/chainbridge/coordinator/1.0.0"

// CoordinatorElectorFactory is used to create multiple instances of CoordinatorElector
// that are using same communication stream
type CoordinatorElectorFactory struct {
	h      host.Host
	comm   comm.Communication
	config relayer.BullyConfig
}

// NewCoordinatorElectorFactory creates new CoordinatorElectorFactory
func NewCoordinatorElectorFactory(h host.Host, config relayer.BullyConfig) *CoordinatorElectorFactory {
	communication := p2p.NewCommunication(h, ProtocolID, h.Peerstore().Peers())

	return &CoordinatorElectorFactory{
		h:      h,
		comm:   communication,
		config: config,
	}
}

// NewCoordinatorElector creates CoordinatorElector for a specific session
// It also starts listening for session specific bully coordination messages.
func (c *CoordinatorElectorFactory) NewCoordinatorElector(sessionID string) *CoordinatorElector {
	bully := &CoordinatorElector{
		sessionID:    sessionID,
		receiveChan:  make(chan *comm.WrappedMessage),
		electionChan: make(chan *comm.WrappedMessage, 1),
		msgChan:      make(chan *comm.WrappedMessage),
		pingChan:     make(chan *comm.WrappedMessage),
		comm:         p2p.NewCommunication(c.h, ProtocolID, c.h.Peerstore().Peers()),
		conf:         c.config,
		hostID:       c.h.ID(),
		mu:           &sync.RWMutex{},
		coordinator:  c.h.ID(),
	}

	go bully.listen()

	return bully
}

// CoordinatorElector is used to execute bully coordinator discovery
type CoordinatorElector struct {
	sessionID    string
	receiveChan  chan *comm.WrappedMessage
	electionChan chan *comm.WrappedMessage
	msgChan      chan *comm.WrappedMessage
	pingChan     chan *comm.WrappedMessage
	comm         comm.Communication
	hostID       peer.ID
	conf         relayer.BullyConfig
	mu           *sync.RWMutex
	coordinator  peer.ID
	sortedPeers  common.SortablePeerSlice
}

// Coordinator starts coordinator discovery using bully algorithm and returns current leader
// Bully coordination is executed on provided peers
func (cc *CoordinatorElector) Coordinator(peers peer.IDSlice) (peer.ID, error) {
	cc.sortedPeers = common.SortPeersForSession(peers, cc.sessionID)

	errChan := make(chan error)
	go cc.startBullyCoordination(errChan)

	select {
	case err := <-errChan:
		return "", err
	case <-time.After(cc.conf.BullyWaitTime):
		break
	}

	return cc.getCoordinator(), nil
}

// listen starts listening for coordinator relevant messages
func (cc *CoordinatorElector) listen() {
	cc.comm.Subscribe(cc.sessionID, comm.CoordinatorPingMsg, cc.msgChan)
	cc.comm.Subscribe(cc.sessionID, comm.CoordinatorElectionMsg, cc.msgChan)
	cc.comm.Subscribe(cc.sessionID, comm.CoordinatorAliveMsg, cc.msgChan)
	cc.comm.Subscribe(cc.sessionID, comm.CoordinatorPingResponseMsg, cc.msgChan)
	cc.comm.Subscribe(cc.sessionID, comm.CoordinatorSelectMsg, cc.msgChan)
	cc.comm.Subscribe(cc.sessionID, comm.CoordinatorLeaveMsg, cc.msgChan)

	for {
		select {
		case msg := <-cc.msgChan:
			switch msg.MessageType {
			case comm.CoordinatorAliveMsg:
				select {
				// waits for confirmation that elector is alive
				case cc.electionChan <- msg:
					break
				case <-time.After(500 * time.Millisecond):
					break
				}
				break
			case comm.CoordinatorSelectMsg:
				cc.receiveChan <- msg
				break
			case comm.CoordinatorElectionMsg:
				cc.receiveChan <- msg
				break
			case comm.CoordinatorPingResponseMsg:
				cc.pingChan <- msg
				break
			case comm.CoordinatorPingMsg:
				cc.comm.Broadcast(
					[]peer.ID{msg.From}, nil, comm.CoordinatorPingResponseMsg, cc.sessionID, nil,
				)
				break
			default:
				break
			}
		}
	}
}

func (cc *CoordinatorElector) elect(errChan chan error) {
	for _, p := range cc.sortedPeers {
		if cc.isPeerIDHigher(p.ID, cc.hostID) {
			cc.comm.Broadcast(peer.IDSlice{p.ID}, nil, comm.CoordinatorElectionMsg, cc.sessionID, errChan)
		}
	}

	select {
	case <-cc.electionChan:
		return
	case <-time.After(cc.conf.ElectionWaitTime):
		cc.setCoordinator(cc.hostID)
		cc.comm.Broadcast(cc.sortedPeers.GetPeerIDs(), []byte{}, comm.CoordinatorSelectMsg, cc.sessionID, errChan)
		return
	}
}

func (cc *CoordinatorElector) startBullyCoordination(errChan chan error) {
	cc.elect(errChan)
	for msg := range cc.receiveChan {
		if msg.MessageType == comm.CoordinatorElectionMsg && !cc.isPeerIDHigher(msg.From, cc.hostID) {
			cc.comm.Broadcast([]peer.ID{msg.From}, []byte{}, comm.CoordinatorAliveMsg, cc.sessionID, errChan)
			cc.elect(errChan)
		} else if msg.MessageType == comm.CoordinatorSelectMsg {
			cc.setCoordinator(msg.From)
		}
	}
}

func (cc *CoordinatorElector) isPeerIDHigher(p1 peer.ID, p2 peer.ID) bool {
	var i1, i2 int
	for i := range cc.sortedPeers {
		if p1 == cc.sortedPeers[i].ID {
			i1 = i
		}
		if p2 == cc.sortedPeers[i].ID {
			i2 = i
		}
	}
	return i1 < i2
}

func (cc *CoordinatorElector) setCoordinator(ID peer.ID) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.isPeerIDHigher(ID, cc.coordinator) || ID == cc.hostID {
		cc.coordinator = ID
	}
}

func (cc *CoordinatorElector) getCoordinator() peer.ID {
	return cc.coordinator
}
