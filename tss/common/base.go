// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package common

import (
	"context"
	"fmt"
	"github.com/ChainSafe/chainbridge-core/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"math/big"
	"runtime/debug"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog"
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
	TID           string
	Party         Party
	PartyStore    map[string]*tss.PartyID
	Communication comm.Communication
	Peers         []peer.ID
	Log           zerolog.Logger

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
func (b *BaseTss) ProcessInboundMessages(ctx context.Context, msgChan chan *comm.WrappedMessage) (err error) {
	ctx, span, logger := observability.CreateSpanAndLoggerFromContext(ctx, "relayer-sygma", "relayer.sygma.tss.BaseTss.ProcessInboundMessages")
	defer span.End()
	defer func() {
		if r := recover(); r != nil {
			_ = observability.LogAndRecordError(&logger, span, fmt.Errorf(string(debug.Stack())), "paniced on ProcessingInboundMessage")
		}
	}()

	for {
		select {
		case wMsg := <-msgChan:
			{
				msg, err := UnmarshalTssMessage(wMsg.Payload)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())
					return err
				}
				observability.LogAndEvent(logger.Debug(), span, "processed inbound message",
					attribute.String("p2pmsg.from", wMsg.From.String()),
					attribute.String("p2pmsg.type", wMsg.MessageType.String()),
					attribute.Bool("p2pmsg.IsBroadcast", msg.IsBroadcast),
					attribute.String("p2pmsg.IsBroadcast", fmt.Sprintf("%x", msg.MsgBytes)),
				)
				ok, err := b.Party.UpdateFromBytes(
					msg.MsgBytes,
					b.PartyStore[wMsg.From.String()],
					msg.IsBroadcast,
					new(big.Int).SetBytes([]byte(b.SID)))
				if !ok {
					return observability.LogAndRecordErrorWithStatus(&logger, span, err, "Not ok updating Party from message Bytes")
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// ProcessOutboundMessages sends messages received from tss out channel to target peers.
// On context cancel stops listening to channel and exits.
func (b *BaseTss) ProcessOutboundMessages(ctx context.Context, outChn chan tss.Message, messageType comm.MessageType) error {
	ctx, span, logger := observability.CreateSpanAndLoggerFromContext(ctx, "relayer-sygma", "relayer.sygma.tss.BaseTss.ProcessOutboundMessages")
	defer span.End()
	for {
		select {
		case msg := <-outChn:
			{
				wireBytes, routing, err := msg.WireBytes()
				if err != nil {
					return observability.LogAndRecordErrorWithStatus(&logger, span, err, "failed to encode recived TSS message bytes")
				}

				msgBytes, err := MarshalTssMessage(wireBytes, routing.IsBroadcast)
				if err != nil {
					return observability.LogAndRecordErrorWithStatus(&logger, span, err, "failed to marshal TSS message")
				}

				peers, err := b.BroadcastPeers(msg)
				if err != nil {
					return observability.LogAndRecordErrorWithStatus(&logger, span, err, "failed to broadcast a message")
				}

				observability.LogAndEvent(logger.Debug(), span, "Processed outbound message",
					attribute.String("p2pmsg.peers", fmt.Sprintf("%s", peers)),
					attribute.String("p2pmsg.type", messageType.String()),
					attribute.Bool("p2pmsg.IsBroadcast", routing.IsBroadcast),
					attribute.String("p2pmsg.full", msg.String()),
				)
				err = b.Communication.Broadcast(ctx, peers, msgBytes, messageType, b.SessionID())
				if err != nil {
					return observability.LogAndRecordErrorWithStatus(&logger, span, err, "error on broadcasting message")
				}
			}
		case <-ctx.Done():
			{
				return nil
			}
		}
	}
}

// BroadcastPeers returns peers that should receive the tss message
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
