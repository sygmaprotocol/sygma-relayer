package elector

import (
	"github.com/ChainSafe/chainbridge-core/comm"
	"github.com/ChainSafe/chainbridge-core/comm/p2p"
	"github.com/ChainSafe/chainbridge-core/config/relayer"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

type CoordinatorElectorType int

const (
	Static CoordinatorElectorType = iota
	Bully
)

const ProtocolID protocol.ID = "/chainbridge/coordinator/1.0.0"

type CoordinatorElector interface {
	Coordinator(peers peer.IDSlice) (peer.ID, error)
}

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
func (c *CoordinatorElectorFactory) CoordinatorElector(
	sessionID string, electorType CoordinatorElectorType,
) CoordinatorElector {
	switch electorType {
	case Static:
		return NewCoordinatorElector(sessionID)
	case Bully:
		return NewBullyCoordinatorElector(sessionID, c.h, c.config, c.comm)
	default:
		return nil
	}
}
