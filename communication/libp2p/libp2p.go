package libp2p

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
	"sync"
)

type Libp2pCommunication struct {
	h                    host.Host
	protocolID           protocol.ID
	streamManager        *StreamManager
	logger               zerolog.Logger
	subscriptionManagers map[ChainBridgeMessageType]*SessionSubscriptionManager
	subscriberLocker     *sync.Mutex
}

func NewCommunication(h host.Host, protocolID protocol.ID) Libp2pCommunication {
	logger := log.With().Str("module", "communication").Str("peer", h.ID().Pretty()).Logger()
	c := Libp2pCommunication{
		h:                    h,
		protocolID:           protocolID,
		streamManager:        NewStreamManager(),
		logger:               logger,
		subscriptionManagers: make(map[ChainBridgeMessageType]*SessionSubscriptionManager),
		subscriberLocker:     &sync.Mutex{},
	}
	c.startProcessingStream()
	return c
}

/** Communication interface methods **/

// Broadcast sends
func (c *Libp2pCommunication) Broadcast(
	peers peer.IDSlice,
	msg []byte,
	msgType ChainBridgeMessageType,
	sessionID comm.SessionID,
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
		c.logger.Error().Err(err).EmbedObject(sessionID).Msg("unable to marshal message")
		return
	}
	c.logger.Debug().EmbedObject(msgType).EmbedObject(sessionID).Msg(
		"broadcasting message",
	)
	for _, p := range peers {
		if hostID != p {
			stream, err := c.h.NewStream(context.TODO(), p, c.protocolID)
			if err != nil {
				c.logger.Error().Err(err).EmbedObject(msgType).EmbedObject(sessionID).Msgf(
					"unable to open stream toward %s", p.Pretty(),
				)
				return
			}

			err = WriteStreamWithBuffer(marshaledMsg, stream)
			if err != nil {
				c.logger.Error().Str("to", string(p)).Err(err).Msg("unable to send message")
				return
			}
			c.logger.Trace().Str(
				"from", string(wMsg.From)).Str(
				"to", p.Pretty()).EmbedObject(
				wMsg.MessageType).EmbedObject(
				wMsg.SessionID).Msg(
				"message sent",
			)
			c.streamManager.AddStream(sessionID, stream)
		}
	}
}

// Subscribe
func (c *Libp2pCommunication) Subscribe(
	msgType ChainBridgeMessageType,
	sessionID comm.SessionID,
	channel chan *comm.WrappedMessage,
) comm.SubscriptionID {
	c.subscriberLocker.Lock()
	defer c.subscriberLocker.Unlock()

	subManager, ok := c.subscriptionManagers[msgType]
	if !ok {
		subManager = NewSessionSubscriptionManager()
		c.subscriptionManagers[msgType] = subManager
	}

	sID := subManager.Subscribe(sessionID, channel)
	c.logger.Info().Str("sessionID", string(sessionID)).Msgf("subscribed to topic %s", msgType)
	return sID
}

// UnSubscribe
func (c *Libp2pCommunication) UnSubscribe(
	msgType ChainBridgeMessageType,
	sessionID comm.SessionID,
	subID comm.SubscriptionID,
) {
	c.subscriberLocker.Lock()
	defer c.subscriberLocker.Unlock()

	subManager, ok := c.subscriptionManagers[msgType]
	if !ok {
		c.logger.Debug().Msgf("cannot find the given channels %s", msgType.String())
		return
	}
	if subManager == nil {
		return
	}

	subManager.UnSubscribe(sessionID, subID)
}

// ReleaseStream
func (c *Libp2pCommunication) ReleaseStream(sessionID comm.SessionID) {
	c.streamManager.ReleaseStream(sessionID)
	c.logger.Info().Str("sessionID", string(sessionID)).Msg("released stream")
}

/** Helper methods **/

func (c Libp2pCommunication) startProcessingStream() {
	c.h.SetStreamHandler(c.protocolID, func(s network.Stream) {
		msg, err := c.processMessageFromStream(s)
		if err != nil {
			c.logger.Error().Err(err).Str("streamID", s.ID()).Msg("unable to process message")
			return
		}

		subscribers := c.getSubscribers(msg.MessageType, msg.SessionID)
		for _, sub := range subscribers {
			sub <- msg
		}
	})
}

func (c *Libp2pCommunication) getSubscribers(
	msgType ChainBridgeMessageType, sessionID comm.SessionID,
) []chan *comm.WrappedMessage {
	c.subscriberLocker.Lock()
	defer c.subscriberLocker.Unlock()

	messageIDSubscriber, ok := c.subscriptionManagers[msgType]
	if !ok {
		c.logger.Debug().Msgf("fail to find subscription manager for message type %s", msgType)
		return nil
	}

	return messageIDSubscriber.GetSubscribers(sessionID)
}

func (c *Libp2pCommunication) processMessageFromStream(s network.Stream) (*comm.WrappedMessage, error) {
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
		"from", string(wrappedMsg.From)).Str(
		"msgType", wrappedMsg.MessageType.String()).Str(
		"sessionID", string(wrappedMsg.SessionID)).Msg(
		"processed message",
	)

	return &wrappedMsg, nil
}
