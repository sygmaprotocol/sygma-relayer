package listener

import (
	"errors"

	"github.com/rs/zerolog/log"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	core_types "github.com/ChainSafe/chainbridge-core/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type DepositHandlers map[message.TransferType]DepositHandlerFunc
type DepositHandlerFunc func(sourceID uint8, destId types.U8, nonce types.U64, resourceID types.Bytes32, calldata []byte) (*message.Message, error)

type SubstrateDepositHandler struct {
	depositHandlers DepositHandlers
}

// NewSubstrateDepositHandler creates an instance of SubstrateDepositHandler that contains
// handler functions for processing deposit events
func NewSubstrateDepositHandler() *SubstrateDepositHandler {
	return &SubstrateDepositHandler{
		depositHandlers: make(map[message.TransferType]DepositHandlerFunc),
	}
}

func (e *SubstrateDepositHandler) HandleDeposit(sourceID uint8, destID types.U8, depositNonce types.U64, resourceID types.Bytes32, calldata []byte, transferType [1]byte) (*message.Message, error) {
	var depositType message.TransferType
	if transferType[0] == 0 {
		depositType = message.FungibleTransfer
	} else {
		return nil, errors.New("no corresponding deposit handler for this transfer type exists")
	}

	depositHandler, err := e.matchTransferTypeHandlerFunc(depositType)
	if err != nil {
		return nil, err
	}

	return depositHandler(sourceID, destID, depositNonce, resourceID, calldata)
}

// matchAddressWithHandlerFunc matches a transfer type with an associated handler function
func (e *SubstrateDepositHandler) matchTransferTypeHandlerFunc(transferType message.TransferType) (DepositHandlerFunc, error) {
	hf, ok := e.depositHandlers[transferType]
	if !ok {
		return nil, errors.New("no corresponding deposit handler for this transfer type exists")
	}
	return hf, nil
}

// RegisterDepositHandler registers an event handler by associating a handler function to a transfer type
func (e *SubstrateDepositHandler) RegisterDepositHandler(transferType message.TransferType, handler DepositHandlerFunc) {
	if transferType == "" {
		return
	}

	log.Info().Msgf("Registered deposit handler for transfer type %s", transferType)
	e.depositHandlers[transferType] = handler
}

//FungibleTransferHandler converts data pulled from event logs into message
// handlerResponse can be an empty slice
func FungibleTransferHandler(sourceID uint8, destId types.U8, nonce types.U64, resourceID types.Bytes32, calldata []byte) (*message.Message, error) {
	if len(calldata) < 84 {
		err := errors.New("invalid calldata length: less than 84 bytes")
		return nil, err
	}

	// amount: first 32 bytes of calldata
	amount := calldata[:32]

	// 32-64 is multiLocation length
	recipientAddressLength, _ := types.IntBytesToBigInt(calldata[32:64])

	// 64 - (64 + multiLocation length) is recipient address
	recipientAddress := calldata[64:(64 + recipientAddressLength.Int64())]

	// if there is priority data, parse it and use it
	payload := []interface{}{
		amount,
		recipientAddress,
	}

	metadata := message.Metadata{}

	return message.NewMessage(uint8(sourceID), uint8(destId), uint64(nonce), core_types.ResourceID(resourceID), message.FungibleTransfer, payload, metadata), nil
}
