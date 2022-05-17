package bully

import (
	"fmt"
	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/ChainSafe/chainbridge-core/communication/p2p"
	"github.com/ChainSafe/chainbridge-core/config/relayer"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"golang.org/x/exp/slices"
	"sync"
	"time"
)

var ProtocolID protocol.ID = "/chainbridge/coordinator/1.0.0"

type BullyCommunicationCoordinator struct {
	h          host.Host
	comm       communication.Communication
	protocolID protocol.ID
	config     relayer.BullyConfig
}

func NewBullyCommunicationCoordinator(h host.Host, sessionID string, config relayer.BullyConfig) BullyCommunicationCoordinator {
	comm := p2p.NewCommunication(h, ProtocolID, h.Peerstore().Peers())

	return BullyCommunicationCoordinator{
		h:          h,
		comm:       comm,
		protocolID: ProtocolID,
		config:     config,
	}
}

// StartBullyCoordination starts bully process to determine dynamic coordinator
// When coordinator is determined it will be sent to coordinatorChan, if error occurs it will be sent to errChan
func (c BullyCommunicationCoordinator) StartBullyCoordination(
	excludedPeers peer.IDSlice, sessionID string,
) Bully {
	pm := NewPeerMap()
	for _, p := range c.h.Peerstore().Peers() {
		if !slices.Contains(excludedPeers, p) {
			pm.Add(p)
		}
	}

	bully := Bully{
		sessionID:    sessionID,
		receiveChan:  make(chan *communication.WrappedMessage),
		electionChan: make(chan *communication.WrappedMessage, 1),
		msgChan:      make(chan *communication.WrappedMessage),
		pingChan:     make(chan *communication.WrappedMessage),
		comm:         c.comm,
		peers:        pm,
		hostID:       c.h.ID(),
		mu:           &sync.RWMutex{},
		coordinator:  c.h.ID(),
	}

	go bully.listen()

	// bully.StartBully(coordinatorChan, errChan)
	return bully
}

type Bully struct {
	sessionID    string
	receiveChan  chan *communication.WrappedMessage
	electionChan chan *communication.WrappedMessage
	msgChan      chan *communication.WrappedMessage
	pingChan     chan *communication.WrappedMessage
	comm         communication.Communication
	peers        Peers
	hostID       peer.ID
	conf         relayer.BullyConfig
	mu           *sync.RWMutex
	coordinator  peer.ID
}

func (b Bully) SetCoordinator(ID peer.ID) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.coordinator = ID
}

func (b Bully) Coordinator() peer.ID {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.coordinator
}

func (b Bully) pingLeader() {
	for {
		coordinator := b.Coordinator()
		// ping only if I am not leader
		if ComparePeerIDs(coordinator, b.hostID) != 0 {
			// b.Send([]peer.ID{coordinator}, p2p.Libp2pBullyMsgPing)
			b.comm.Broadcast([]peer.ID{coordinator}, []byte{}, communication.CoordinatorPingMsg, b.sessionID, nil)
			select {
			// wait for ping response
			case <-b.pingChan:
				break
			// end leader if not responding after PingWaitTime
			case <-time.After(b.conf.PingWaitTime):
				b.msgChan <- &communication.WrappedMessage{
					MessageType: communication.CoordinatorLeaveMsg,
					SessionID:   "1",
					Payload:     nil,
					From:        coordinator,
				}
				time.Sleep(b.conf.PingBackOff)
				break
			}
		}
		time.Sleep(b.conf.PingInterval)
	}
}

func (b Bully) listen() {
	b.comm.Subscribe(b.sessionID, communication.CoordinatorPingMsg, b.msgChan)
	b.comm.Subscribe(b.sessionID, communication.CoordinatorElectionMsg, b.msgChan)
	b.comm.Subscribe(b.sessionID, communication.CoordinatorAliveMsg, b.msgChan)
	b.comm.Subscribe(b.sessionID, communication.CoordinatorPingResponseMsg, b.msgChan)
	b.comm.Subscribe(b.sessionID, communication.CoordinatorSelectMsg, b.msgChan)
	b.comm.Subscribe(b.sessionID, communication.CoordinatorLeaveMsg, b.msgChan)

	for {
		select {
		case msg := <-b.msgChan:
			switch msg.MessageType {
			case communication.CoordinatorLeaveMsg:
				//if msg.From.Pretty() == b.coordinator.Pretty() {
				//	b.peers.Delete(msg.From)
				//	b.SetCoordinator(b.ID)
				//	b.Elect()
				//}
				break
			case communication.CoordinatorAliveMsg:
				select {
				case b.electionChan <- msg:
					break
				case <-time.After(200 * time.Millisecond):
					break
				}
				break
			case communication.CoordinatorSelectMsg:
				b.receiveChan <- msg
				fmt.Println("processed coordinator")
				break
			case communication.CoordinatorElectionMsg:
				b.receiveChan <- msg
				break
			case communication.CoordinatorPingResponseMsg:
				b.pingChan <- msg
				break
			case communication.CoordinatorPingMsg:
				b.comm.Broadcast(
					[]peer.ID{msg.From}, nil, communication.CoordinatorPingResponseMsg, b.sessionID, nil,
				)
				break
			default:
				break
			}
		}
	}
}

func (b Bully) elect(coordinatorChan chan peer.ID, errChan chan error) {
	for _, p := range b.peers.PeerData() {
		if ComparePeerIDs(p, b.hostID) == 1 {
			b.comm.Broadcast(peer.IDSlice{p}, nil, communication.CoordinatorElectionMsg, b.sessionID, nil)
		}
	}

	select {
	case <-b.electionChan:
		return
	case <-time.After(b.conf.ElectionWaitTime):
		// coordinatorChan <- b.hostID
		b.SetCoordinator(b.hostID)
		b.comm.Broadcast(b.peers.PeerData(), []byte{}, communication.CoordinatorSelectMsg, b.sessionID, errChan)
		return
	}
}

func (b Bully) StartBully(coordinatorChan chan peer.ID, errChan chan error) {
	b.elect(coordinatorChan, errChan)
	// go b.pingLeader()
	for msg := range b.receiveChan {
		// fmt.Printf("message %v", msg)
		if msg.MessageType == communication.CoordinatorElectionMsg && ComparePeerIDs(msg.From, b.hostID) == -1 {
			b.comm.Broadcast(b.peers.PeerData(), []byte{}, communication.CoordinatorAliveMsg, b.sessionID, errChan)
			b.elect(coordinatorChan, errChan)
		} else if msg.MessageType == communication.CoordinatorSelectMsg {
			coordinatorChan <- msg.From
		}
	}
}
