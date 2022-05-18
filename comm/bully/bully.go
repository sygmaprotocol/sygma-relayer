package bully

import (
	"fmt"
	"github.com/ChainSafe/chainbridge-core/comm"
	"github.com/ChainSafe/chainbridge-core/comm/p2p"
	"github.com/ChainSafe/chainbridge-core/config/relayer"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"golang.org/x/exp/slices"
	"sync"
	"time"
)

const ProtocolID protocol.ID = "/chainbridge/coordinator/1.0.0"

// CommunicationCoordinatorFactory todo
type CommunicationCoordinatorFactory struct {
	h          host.Host
	comm       comm.Communication
	protocolID protocol.ID
	config     relayer.BullyConfig
}

// NewCommunicationCoordinatorFactory todo
func NewCommunicationCoordinatorFactory(h host.Host, config relayer.BullyConfig) CommunicationCoordinatorFactory {
	communication := p2p.NewCommunication(h, ProtocolID, h.Peerstore().Peers())

	return CommunicationCoordinatorFactory{
		h:          h,
		comm:       communication,
		protocolID: ProtocolID,
		config:     config,
	}
}

// CommunicationCoordinator
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
	allPeers     peer.IDSlice
}

// NewCommunicationCoordinator TODO
func (c CommunicationCoordinatorFactory) NewCommunicationCoordinator(
	sessionID string,
) *CommunicationCoordinator {
	bully := &CommunicationCoordinator{
		sessionID:    sessionID,
		receiveChan:  make(chan *comm.WrappedMessage),
		electionChan: make(chan *comm.WrappedMessage, 1),
		msgChan:      make(chan *comm.WrappedMessage),
		pingChan:     make(chan *comm.WrappedMessage),
		comm:         c.comm,
		hostID:       c.h.ID(),
		mu:           &sync.RWMutex{},
		coordinator:  c.h.ID(),
		allPeers:     c.h.Peerstore().Peers(),
	}

	go bully.listen()

	return bully
}

func (b *CommunicationCoordinator) GetCoordinator(excludedPeers peer.IDSlice) (peer.ID, error) {
	b.sortedPeers = common.SortPeersForSession(b.getPeers(excludedPeers), b.sessionID)

	errChan := make(chan error)
	go b.startBullyCoordination(errChan)

	select {
	case err := <-errChan:
		return "", err
	case <-time.After(b.conf.BullyWaitTime):
		break
	}

	return b.getCoordinator(), nil
}

// listen starts listening for coordinator relevant messages
func (b *CommunicationCoordinator) listen() {
	b.comm.Subscribe(b.sessionID, comm.CoordinatorPingMsg, b.msgChan)
	b.comm.Subscribe(b.sessionID, comm.CoordinatorElectionMsg, b.msgChan)
	b.comm.Subscribe(b.sessionID, comm.CoordinatorAliveMsg, b.msgChan)
	b.comm.Subscribe(b.sessionID, comm.CoordinatorPingResponseMsg, b.msgChan)
	b.comm.Subscribe(b.sessionID, comm.CoordinatorSelectMsg, b.msgChan)
	b.comm.Subscribe(b.sessionID, comm.CoordinatorLeaveMsg, b.msgChan)

	for {
		select {
		case msg := <-b.msgChan:
			switch msg.MessageType {
			case comm.CoordinatorLeaveMsg:
				//if msg.From.Pretty() == b.coordinator.Pretty() {
				//	b.peers.Delete(msg.From)
				//	b.setCoordinator(b.ID)
				//	b.Elect()
				//}
				fmt.Println("SHOULD NOT HAPPEN")
				break
			case comm.CoordinatorAliveMsg:
				select {
				case b.electionChan <- msg:
					break
				case <-time.After(200 * time.Millisecond):
					break
				}
				break
			case comm.CoordinatorSelectMsg:
				b.receiveChan <- msg
				break
			case comm.CoordinatorElectionMsg:
				b.receiveChan <- msg
				break
			case comm.CoordinatorPingResponseMsg:
				b.pingChan <- msg
				break
			case comm.CoordinatorPingMsg:
				b.comm.Broadcast(
					[]peer.ID{msg.From}, nil, comm.CoordinatorPingResponseMsg, b.sessionID, nil,
				)
				break
			default:
				break
			}
		}
	}
}

func (b *CommunicationCoordinator) elect(errChan chan error) {
	for _, p := range b.sortedPeers {
		if b.isGreater(p.ID, b.hostID) {
			b.comm.Broadcast(peer.IDSlice{p.ID}, nil, comm.CoordinatorElectionMsg, b.sessionID, errChan)
		}
	}

	select {
	case <-b.electionChan:
		return
	case <-time.After(b.conf.ElectionWaitTime):
		b.setCoordinator(b.hostID)
		b.comm.Broadcast(b.sortedPeers.GetPeerIDs(), []byte{}, comm.CoordinatorSelectMsg, b.sessionID, errChan)
		return
	}
}

func (b *CommunicationCoordinator) startBullyCoordination(errChan chan error) {
	b.elect(errChan)
	// go b.pingLeader()
	for msg := range b.receiveChan {
		if msg.MessageType == comm.CoordinatorElectionMsg && !b.isGreater(msg.From, b.hostID) {
			b.comm.Broadcast([]peer.ID{msg.From}, []byte{}, comm.CoordinatorAliveMsg, b.sessionID, errChan)
			b.elect(errChan)
		} else if msg.MessageType == comm.CoordinatorSelectMsg {
			// fmt.Printf("[%s] SET COORDINATOR:%s\n", b.hostID.Pretty(), msg.From.Pretty())
			b.setCoordinator(msg.From)
		}
	}
}

// getPeers returns all peers that are not excluded
func (b *CommunicationCoordinator) getPeers(excludedPeers peer.IDSlice) peer.IDSlice {
	peers := peer.IDSlice{}
	for _, p := range b.allPeers {
		if !slices.Contains(excludedPeers, p) {
			peers = append(peers, p)
		}
	}
	return peers
}

// isGreater returns true if p1 > p2
func (b *CommunicationCoordinator) isGreater(p1 peer.ID, p2 peer.ID) bool {
	var i1, i2 int
	for i := range b.sortedPeers {
		if p1 == b.sortedPeers[i].ID {
			i1 = i
		}
		if p2 == b.sortedPeers[i].ID {
			i2 = i
		}
	}
	return i1 < i2
}

func (b *CommunicationCoordinator) setCoordinator(ID peer.ID) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.coordinator = ID
}

func (b *CommunicationCoordinator) getCoordinator() peer.ID {
	return b.coordinator
}

func (b *CommunicationCoordinator) pingLeader() {
	for {
		coordinator := b.getCoordinator()
		// ping only if I am not leader
		if coordinator != b.hostID {
			// b.Send([]peer.ID{coordinator}, p2p.Libp2pBullyMsgPing)
			b.comm.Broadcast([]peer.ID{coordinator}, []byte{}, comm.CoordinatorPingMsg, b.sessionID, nil)
			select {
			// wait for ping response
			case <-b.pingChan:
				break
			// end leader if not responding after PingWaitTime
			case <-time.After(b.conf.PingWaitTime):
				b.msgChan <- &comm.WrappedMessage{
					MessageType: comm.CoordinatorLeaveMsg,
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
