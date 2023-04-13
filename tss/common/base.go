// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"context"
	"math/big"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog"
	"github.com/sourcegraph/conc/panics"
	"github.com/sourcegraph/conc/pool"
)

type Party interface {
	UpdateFromBytes(wireBytes []byte, from *tss.PartyID, isBroadcast bool, sessionID *big.Int) (bool, *tss.Error)
	Start() *tss.Error
	WaitingFor() []*tss.PartyID
}

// BaseTss contains common variables and methods to
// all tss processes.
type BaseTss struct {
	Host          host.Host
	SID           string
	Party         Party
	PartyStore    map[string]*tss.PartyID
	Communication comm.Communication
	Peers         []peer.ID
	Log           zerolog.Logger

	ErrChn chan error
	Cancel context.CancelFunc
}

// PopulatePartyStore populates party store map with sorted parties for
// mapping message senders to parties
func (b *BaseTss) PopulatePartyStore(parties tss.SortedPartyIDs) {
	for _, party := range parties {
		b.PartyStore[party.Id] = party
	}
}

// ProcessInboundMessages processes messages from tss parties and updates local party accordingly.
func (b *BaseTss) ProcessInboundMessages(p *pool.ContextPool, msgChan chan *comm.WrappedMessage) {
	p.Go(func(ctx context.Context) error {
		for {
			select {
			case wMsg := <-msgChan:
				{
					b.Log.Debug().Msgf("processed inbound message from %s", wMsg.From)

					msg, err := UnmarshalTssMessage(wMsg.Payload)
					if err != nil {
						b.ErrChn <- err
						return err
					}

					var pc panics.Catcher
					pc.Try(func() {
						ok, err := b.Party.UpdateFromBytes(
							msg.MsgBytes,
							b.PartyStore[wMsg.From.Pretty()],
							msg.IsBroadcast,
							new(big.Int).SetBytes([]byte(b.SID)))
						if !ok {
							panic(err)
						}
					})
					if pc.Recovered().AsError() != nil {
						return err
					}
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
}

// ProcessOutboundMessages sends messages received from tss out channel to target peers.
// On context cancel stops listening to channel and exits.
func (b *BaseTss) ProcessOutboundMessages(p *pool.ContextPool, outChn chan tss.Message, messageType comm.MessageType) {
	p.Go(func(ctx context.Context) error {
		for {
			select {
			case msg := <-outChn:
				{
					b.Log.Debug().Msg(msg.String())
					wireBytes, routing, err := msg.WireBytes()
					if err != nil {
						return err
					}

					msgBytes, err := MarshalTssMessage(wireBytes, routing.IsBroadcast)
					if err != nil {
						return err
					}

					peers, err := b.BroadcastPeers(msg)
					if err != nil {
						return err
					}

					b.Log.Debug().Msgf("sending message to %s", peers)
					p.Go(func(ctx context.Context) error {
						b.Communication.Broadcast(peers, msgBytes, messageType, b.SessionID(), b.ErrChn)
						return nil
					})
				}
			case <-ctx.Done():
				{
					return ctx.Err()
				}
			}
		}
	})
}

// BroccastPeers returns peers that should receive the tss message
func (b *BaseTss) BroadcastPeers(msg tss.Message) ([]peer.ID, error) {
	if msg.IsBroadcast() {
		return b.Peers, nil
	} else {
		return PeersFromParties(msg.GetTo())
	}
}

func (b *BaseTss) SessionID() string {
	return b.SID
}
