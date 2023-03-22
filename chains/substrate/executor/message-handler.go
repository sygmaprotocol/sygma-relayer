package executor

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ethereum/go-ethereum/common"

	"github.com/rs/zerolog/log"
)

type Handlers map[message.TransferType]MessageHandlerFunc
type MessageHandlerFunc func(m *message.Message) (*chains.Proposal, error)

type SubstrateMessageHandler struct {
	handlers Handlers
}

// NewSubstrateMessageHandler creates an instance of SubstrateMessageHandler that contains
// message handler functions for converting deposit message into a chain specific
// proposal
func NewSubstrateMessageHandler() *SubstrateMessageHandler {
	return &SubstrateMessageHandler{
		handlers: make(map[message.TransferType]MessageHandlerFunc),
	}
}

func (mh *SubstrateMessageHandler) HandleMessage(m *message.Message) (*chains.Proposal, error) {
	// Based on handler that was registered on BridgeContract
	handleMessage, err := mh.matchTransferTypeHandlerFunc(m.Type)
	if err != nil {
		return nil, err
	}
	log.Info().Str("type", string(m.Type)).Uint8("src", m.Source).Uint8("dst", m.Destination).Uint64("nonce", m.DepositNonce).Str("resourceID", fmt.Sprintf("%x", m.ResourceId)).Msg("Handling new message")
	prop, err := handleMessage(m)
	if err != nil {
		return nil, err
	}
	return prop, nil
}

func (mh *SubstrateMessageHandler) matchTransferTypeHandlerFunc(transferType message.TransferType) (MessageHandlerFunc, error) {
	h, ok := mh.handlers[transferType]
	if !ok {
		return nil, fmt.Errorf("no corresponding message handler for this transfer type %s exists", transferType)
	}
	return h, nil
}

// RegisterEventHandler registers an message handler by associating a handler function to a specified transfer type
func (mh *SubstrateMessageHandler) RegisterMessageHandler(transferType message.TransferType, handler MessageHandlerFunc) {
	if transferType == "" {
		return
	}

	log.Info().Msgf("Registered message handler for transfer type %s", transferType)

	mh.handlers[transferType] = handler
}

func FungibleTransferMessageHandler(m *message.Message) (*chains.Proposal, error) {
	if len(m.Payload) != 2 {
		return nil, errors.New("malformed payload. Len  of payload should be 2")
	}
	amount, ok := m.Payload[0].([]byte)
	if !ok {
		return nil, errors.New("wrong payload amount format")
	}
	recipient, ok := m.Payload[1].([]byte)
	if !ok {
		return nil, errors.New("wrong payload recipient format")
	}
	var data []byte
	data = append(data, common.LeftPadBytes(amount, 32)...) // amount (uint256)

	recipientLen := big.NewInt(int64(len(recipient))).Bytes()
	data = append(data, common.LeftPadBytes(recipientLen, 32)...)
	data = append(data, recipient...)
	return chains.NewProposal(m.Source, m.Destination, m.DepositNonce, m.ResourceId, data, m.Metadata), nil
}
