package common

import (
	"context"
	"errors"

	"github.com/ChainSafe/sygma/comm"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog"
)

type Party interface {
	UpdateFromBytes(wireBytes []byte, from *tss.PartyID, isBroadcast bool) (bool, *tss.Error)
	Start() *tss.Error
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
func (b *BaseTss) ProcessInboundMessages(ctx context.Context, msgChan chan *comm.WrappedMessage) {
	for {
		select {
		case wMsg := <-msgChan:
			{
				go func() {
					defer func() {
						if r := recover(); r != nil {
							switch x := r.(type) {
							case string:
								b.ErrChn <- errors.New(x)
							case error:
								b.ErrChn <- x
							default:
								b.ErrChn <- errors.New("unknown panic")
							}
						}
					}()
					b.Log.Debug().Msgf("processed inbound message from %s", wMsg.From)

					msg, err := UnmarshalTssMessage(wMsg.Payload)
					if err != nil {
						b.ErrChn <- err
						return
					}

					ok, err := b.Party.UpdateFromBytes(msg.MsgBytes, b.PartyStore[wMsg.From.Pretty()], msg.IsBroadcast)
					if !ok {
						b.ErrChn <- err
					}
				}()
			}
		case <-ctx.Done():
			{
				return
			}
		}
	}
}

// ProcessOutboundMessages sends messages received from tss out channel to target peers.
// On context cancel stops listening to channel and exits.
func (b *BaseTss) ProcessOutboundMessages(ctx context.Context, outChn chan tss.Message, messageType comm.MessageType) {
	for {
		select {
		case msg := <-outChn:
			{
				b.Log.Debug().Msg(msg.String())
				wireBytes, routing, err := msg.WireBytes()
				if err != nil {
					b.ErrChn <- err
					return
				}

				msgBytes, err := MarshalTssMessage(wireBytes, routing.IsBroadcast)
				if err != nil {
					b.ErrChn <- err
					return
				}

				peers, err := b.BroadcastPeers(msg)
				if err != nil {
					b.ErrChn <- err
					return
				}

				b.Log.Debug().Msgf("sending message to %s", peers)
				go b.Communication.Broadcast(peers, msgBytes, messageType, b.SessionID(), b.ErrChn)
			}
		case <-ctx.Done():
			{
				return
			}
		}
	}
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
