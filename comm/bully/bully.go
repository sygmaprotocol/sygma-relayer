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
func (c *CommunicationCoordinatorFactory) NewCommunicationCoordinator(
	sessionID string, names map[peer.ID]string,
) *CommunicationCoordinator {
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
		allPeers:     c.h.Peerstore().Peers(),
	}

	go bully.listen(names)

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
	allPeers     peer.IDSlice
}

// GetCoordinator starts coordinator discovery using bully algorithm and returns current leader
func (cc *CommunicationCoordinator) GetCoordinator(excludedPeers peer.IDSlice, names map[peer.ID]string) (peer.ID, error) {
	cc.sortedPeers = common.SortPeersForSession(cc.getPeers(excludedPeers), cc.sessionID)

	errChan := make(chan error)
	go cc.startBullyCoordination(errChan, names)

	select {
	case err := <-errChan:
		return "", err
	case <-time.After(cc.conf.BullyWaitTime):
		break
	}

	return cc.getCoordinator(), nil
}

// listen starts listening for coordinator relevant messages
func (cc *CommunicationCoordinator) listen(names map[peer.ID]string) {
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
				case cc.electionChan <- msg:
					break
				case <-time.After(500 * time.Millisecond):
					fmt.Printf("%s THIS IS HAPPENING!!\n", names[cc.hostID])
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

func (cc *CommunicationCoordinator) elect(errChan chan error, names map[peer.ID]string) {
	for _, p := range cc.sortedPeers {
		if cc.isPeerIDHigher(p.ID, cc.hostID) {
			cc.comm.Broadcast(peer.IDSlice{p.ID}, nil, comm.CoordinatorElectionMsg, cc.sessionID, errChan)
		}
	}

	select {
	case msg := <-cc.electionChan:
		fmt.Printf("%s -> %s LUCKILY HE IS ALIVE\n", names[msg.From], names[cc.hostID])
		return
	case <-time.After(cc.conf.ElectionWaitTime):
		fmt.Printf("%s I SET MYSELF AS COORDINATOR!?\n", names[cc.hostID])
		cc.setCoordinator(cc.hostID)
		cc.comm.Broadcast(cc.sortedPeers.GetPeerIDs(), []byte{}, comm.CoordinatorSelectMsg, cc.sessionID, errChan)
		return
	}
}

func (cc *CommunicationCoordinator) startBullyCoordination(errChan chan error, names map[peer.ID]string) {
	cc.elect(errChan, names)
	for msg := range cc.receiveChan {
		if msg.MessageType == comm.CoordinatorElectionMsg && !cc.isPeerIDHigher(msg.From, cc.hostID) {
			fmt.Printf("%s -> %s CHECKS IF IS ALIVE?\n", names[msg.From], names[cc.hostID])
			cc.comm.Broadcast([]peer.ID{msg.From}, []byte{}, comm.CoordinatorAliveMsg, cc.sessionID, errChan)
			cc.elect(errChan, names)
		} else if msg.MessageType == comm.CoordinatorSelectMsg {
			fmt.Printf("%s -> %s TOLD ME THAT HE IS COORDINATOR!\n", names[msg.From], names[cc.hostID])
			cc.setCoordinator(msg.From)
		}
	}
}

// getPeers returns all peers that are not excluded
func (cc *CommunicationCoordinator) getPeers(excludedPeers peer.IDSlice) peer.IDSlice {
	peers := peer.IDSlice{}
	for _, p := range cc.allPeers {
		if !slices.Contains(excludedPeers, p) {
			peers = append(peers, p)
		}
	}
	return peers
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
