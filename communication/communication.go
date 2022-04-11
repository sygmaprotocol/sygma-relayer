package communication

import (
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"strconv"
	"strings"
	"time"
)

// WrappedMessage is a message with type in it
type WrappedMessage struct {
	MessageType ChainBridgeMessageType `json:"message_type"`
	SessionID   string                 `json:"message_id"`
	Payload     []byte                 `json:"payload"`
	From        peer.ID                `json:"from"`
}

// Communication
type Communication interface {
	// Broadcast sends message to provided peers
	Broadcast(peers peer.IDSlice, msg []byte, msgType ChainBridgeMessageType, sessionID string)
	// Subscribe subscribes provided channel to a specific message type for a provided session
	// Returns SubscriptionID - unique identifier of created subscription that is used to unsubscribe from subscription
	Subscribe(sessionID string, msgType ChainBridgeMessageType, channel chan *WrappedMessage) SubscriptionID
	// UnSubscribe unsuscribes from subscription defined by provided SubscriptionID
	UnSubscribe(subID SubscriptionID)
}

// SubscriptionID is unique identifier for each subscription
// It is defined as: SessionID-MessageType-SubscriptionIdentifier
type SubscriptionID string

func NewSubscriptionID(sessionID string, msgType ChainBridgeMessageType) SubscriptionID {
	return SubscriptionID(fmt.Sprintf("%s-%d-%d", sessionID, msgType, uint32(time.Now().UnixNano())))
}

func (sID SubscriptionID) SessionID() string {
	sessID, _, _, err := sID.Unwrap()
	if err != nil {
		return ""
	}
	return sessID
}

func (sID SubscriptionID) MessageType() ChainBridgeMessageType {
	_, msgType, _, err := sID.Unwrap()
	if err != nil {
		return msgType
	}
	return msgType
}

func (sID SubscriptionID) SubscriptionIdentifier() string {
	_, _, subID, err := sID.Unwrap()
	if err != nil {
		return ""
	}
	return subID
}

func (sID SubscriptionID) Unwrap() (string, ChainBridgeMessageType, string, error) {
	subIDParts := strings.Split(string(sID), "-")
	if len(subIDParts) != 3 {
		return "", Unknown, "", errors.New("invalid subscriptionID")
	}

	msgType, err := strconv.ParseInt(subIDParts[1], 10, 8)
	if err != nil {
		return "", Unknown, "", err
	}

	return subIDParts[0], ChainBridgeMessageType(msgType), subIDParts[2], nil
}
