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

// CommunicationCoordinatorFactory is used to create multiple instances of CommunicationCoordinator
// that are using same communication stream
type CommunicationCoordinatorFactory struct {
	h      host.Host
	comm   comm.Communication
	config relayer.BullyConfig
}

// NewCommunicationCoordinatorFactory creates new CommunicationCoordinatorFactory
func NewCommunicationCoordinatorFactory(h host.Host, config relayer.BullyConfig) *CommunicationCoordinatorFactory {
	communication := p2p.NewCommunication(h, ProtocolID, h.Peerstore().Peers())

	return &CommunicationCoordinatorFactory{
		h:      h,
		comm:   communication,
		config: config,
	}
}

// NewCommunicationCoordinator creates CommunicationCoordinator for a specific session
// It also starts listening for session specific bully coordination messages.
func (c *CommunicationCoordinatorFactory) NewCommunicationCoordinator(sessionID string) *CommunicationCoordinator {
	bully := &CommunicationCoordinator{
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

// CommunicationCoordinator is used to execute bully coordinator discovery
type CommunicationCoordinator struct {
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
func (cc *CommunicationCoordinator) Coordinator(peers peer.IDSlice) (peer.ID, error) {
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
func (cc *CommunicationCoordinator) listen() {
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

func (cc *CommunicationCoordinator) elect(errChan chan error) {
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

func (cc *CommunicationCoordinator) startBullyCoordination(errChan chan error) {
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

func (cc *CommunicationCoordinator) isPeerIDHigher(p1 peer.ID, p2 peer.ID) bool {
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

func (cc *CommunicationCoordinator) setCoordinator(ID peer.ID) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.isPeerIDHigher(ID, cc.coordinator) || ID == cc.hostID {
		cc.coordinator = ID
	}
}

func (cc *CommunicationCoordinator) getCoordinator() peer.ID {
	return cc.coordinator
}
