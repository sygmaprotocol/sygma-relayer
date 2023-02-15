package executor

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"unsafe"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/executor/proposal"
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/rs/zerolog/log"
)

type Handlers map[message.TransferType]MessageHandlerFunc
type MessageHandlerFunc func(m *message.Message) (*proposal.Proposal, error)

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

func (mh *SubstrateMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
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

var substratePK = signature.KeyringPair{
	URI:       "//Alice",
	PublicKey: []byte{0xd4, 0x35, 0x93, 0xc7, 0x15, 0xfd, 0xd3, 0x1c, 0x61, 0x14, 0x1a, 0xbd, 0x4, 0xa9, 0x9f, 0xd6, 0x82, 0x2c, 0x85, 0x58, 0x85, 0x4c, 0xcd, 0xe3, 0x9a, 0x56, 0x84, 0xe7, 0xa5, 0x6d, 0xa2, 0x7d},
	Address:   "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
}

func FungibleTransferMessageHandler(m *message.Message) (*proposal.Proposal, error) {
	if len(m.Payload) != 2 {
		return nil, errors.New("malformed payload. Len  of payload should be 2")
	}
	amount, ok := m.Payload[0].([]byte)
	if !ok {
		return nil, errors.New("wrong payload amount format")
	}
	reciever, ok := m.Payload[1].([]byte)
	if !ok {
		return nil, errors.New("wrong payload recipient format")
	}
	var data []byte
	data = append(data, common.LeftPadBytes(amount, 32)...) // amount (uint256)
	acc := *(*[]types.U8)(unsafe.Pointer(&reciever))
	recipient := types.MultiLocationV1{
		Parents: 0,
		Interior: types.JunctionsV1{
			IsX1: true,
			X1: types.JunctionV1{
				IsAccountID32: true,
				AccountID32NetworkID: types.NetworkID{
					IsAny: true,
				},
				AccountID: acc,
			},
		},
	}

	bt := bytes.NewBuffer([]byte{})
	encoder := scale.NewEncoder(bt)
	_ = recipient.Encode(*encoder)

	bites := bt.Bytes()
	var rec []byte
	rec = append(rec, bites[:4]...) // recipient ([]byte)
	rec = append(rec, bites[5:]...) // recipient ([]byte)

	recipientLen := big.NewInt(int64(len(bt.Bytes())) - 1).Bytes()
	data = append(data, common.LeftPadBytes(recipientLen, 32)...)
	data = append(data, rec...) // recipient ([]byte)
	return proposal.NewProposal(m.Source, m.Destination, m.DepositNonce, m.ResourceId, data), nil
}
