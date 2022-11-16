// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"github.com/ChainSafe/chainbridge-core/relayer/message"
)

type EventHandler interface {
	HandleEvents(evtI interface{}, msgChan chan []*message.Message) error
}
type NonFungibleTransfer struct {
	eventHandler EventHandler
}

func NewNonFungibleTransferEventHandler(eventhandler EventHandler) *NonFungibleTransfer {
	return &NonFungibleTransfer{
		eventHandler: eventhandler,
	}
}
func (eh *NonFungibleTransfer) HandleEvents(evtI interface{}, msgChan chan []*message.Message) {}

type GenericTransfer struct {
	eventHandler EventHandler
}

func NewGenericTransferEventHandler(eventhandler EventHandler) *GenericTransfer {
	return &GenericTransfer{
		eventHandler: eventhandler,
	}
}
func (eh *GenericTransfer) HandleEvents(evtI interface{}, msgChan chan []*message.Message) {}

type FungibleTransfer struct {
	eventHandler EventHandler
}

func NewFungibleTransferEventHandler(eventhandler EventHandler) *FungibleTransfer {
	return &FungibleTransfer{
		eventHandler: eventhandler,
	}
}
func (eh *FungibleTransfer) HandleEvents(evtI interface{}, msgChan chan []*message.Message) {

}
