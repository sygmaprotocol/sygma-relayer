package p2p

import (
	"context"
	"encoding/json"
	comm "github.com/ChainSafe/chainbridge-core/communication"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Libp2pCommunication struct {
	SessionSubscriptionManager
	h             host.Host
	protocolID    protocol.ID
	streamManager *StreamManager
	logger        zerolog.Logger
}

func NewCommunication(h host.Host, protocolID protocol.ID) comm.Communication {
	logger := log.With().Str("Module", "communication").Str("Peer", h.ID().Pretty()).Logger()
	c := Libp2pCommunication{
		SessionSubscriptionManager: NewSessionSubscriptionManager(),
		h:                          h,
		protocolID:                 protocolID,
		streamManager:              NewStreamManager(),
		logger:                     logger,
	}
	// start processing incoming messages
	c.h.SetStreamHandler(c.protocolID, c.streamHandlerFunc)
	return c
}

/** Communication interface methods **/

func (c Libp2pCommunication) Broadcast(
	peers peer.IDSlice,
	msg []byte,
	msgType comm.ChainBridgeMessageType,
	sessionID string,
) {
	hostID := c.h.ID()
	wMsg := comm.WrappedMessage{
		MessageType: msgType,
		SessionID:   sessionID,
		Payload:     msg,
		From:        hostID,
	}
	marshaledMsg, err := json.Marshal(wMsg)
	if err != nil {
		c.logger.Error().Err(err).Str("SessionID", sessionID).Msg("unable to marshal message")
		return
	}
	c.logger.Debug().Str("MsgType", msgType.String()).Str("SessionID", sessionID).Msg(
		"broadcasting message",
	)
	for _, p := range peers {
		if hostID != p {
			stream, err := c.h.NewStream(context.TODO(), p, c.protocolID)
			if err != nil {
				c.logger.Error().Err(err).Str("MsgType", msgType.String()).Str("SessionID", sessionID).Msgf(
					"unable to open stream toward %s", p.Pretty(),
				)
				return
			}

			err = WriteStreamWithBuffer(marshaledMsg, stream)
			if err != nil {
				c.logger.Error().Str("To", string(p)).Err(err).Msg("unable to send message")
				return
			}
			c.logger.Trace().Str(
				"From", string(wMsg.From)).Str(
				"To", p.Pretty()).Str(
				"MsgType", msgType.String()).Str(
				"SessionID", sessionID).Msg(
				"message sent",
			)
			c.streamManager.AddStream(sessionID, stream)
		}
	}
}

func (c Libp2pCommunication) Subscribe(
	sessionID string,
	msgType comm.ChainBridgeMessageType,
	channel chan *comm.WrappedMessage,
) comm.SubscriptionID {
	subID := c.subscribe(sessionID, msgType, channel)
	// c.logger.Info().Str("SessionID", sessionID).Msgf("subscribed to message type %s", msgType)
	return subID
}

func (c Libp2pCommunication) UnSubscribe(
	subID comm.SubscriptionID,
) {
	c.unSubscribe(subID)
	// c.logger.Info().Str("SessionID", sessionID).Msgf("subscribed to message type %s", msgType)
}

// TODO - is needed ?
func (c Libp2pCommunication) EndSession(sessionID string) {
	c.streamManager.ReleaseStream(sessionID)
	c.logger.Info().Str("SessionID", sessionID).Msg("released stream")
}

/** Helper methods **/

func (c Libp2pCommunication) streamHandlerFunc(s network.Stream) {
	msg, err := c.processMessageFromStream(s)
	if err != nil {
		c.logger.Error().Err(err).Str("StreamID", s.ID()).Msg("unable to process message")
		return
	}

	subscribers := c.getSubscribers(msg.SessionID, msg.MessageType)
	for _, sub := range subscribers {
		sub := sub
		go func() {
			sub <- msg
		}()
	}
}

func (c Libp2pCommunication) processMessageFromStream(s network.Stream) (*comm.WrappedMessage, error) {
	msgBytes, err := ReadStreamWithBuffer(s)
	if err != nil {
		c.streamManager.AddStream("UNKNOWN", s)
		return nil, err
	}

	var wrappedMsg comm.WrappedMessage
	if err := json.Unmarshal(msgBytes, &wrappedMsg); nil != err {
		c.streamManager.AddStream("UNKNOWN", s)
		return nil, err
	}

	c.streamManager.AddStream(wrappedMsg.SessionID, s)

	c.logger.Trace().Str(
		"From", string(wrappedMsg.From)).Str(
		"MsgType", wrappedMsg.MessageType.String()).Str(
		"SessionID", wrappedMsg.SessionID).Msg(
		"processed message",
	)

	return &wrappedMsg, nil
}
