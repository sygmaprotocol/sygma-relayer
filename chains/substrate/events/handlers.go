// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"github.com/ChainSafe/chainbridge-core/relayer/message"
)

type NonFungibleTransfer struct {
}

func NewNonFungibleTransferEventHandler() *NonFungibleTransfer {
	return &NonFungibleTransfer{}
}
func (eh *NonFungibleTransfer) HandleEvents(evtI interface{}, msgChan chan []*message.Message) {}

type GenericTransfer struct{}

func NewGenericTransferEventHandler() *GenericTransfer {
	return &GenericTransfer{}
}
func (eh *GenericTransfer) HandleEvents(evtI interface{}, msgChan chan []*message.Message) {}

type FungibleTransfer struct{}

func NewFungibleTransferEventHandler() *FungibleTransfer {
	return &FungibleTransfer{}
}
func (eh *FungibleTransfer) HandleEvents(evtI interface{}, msgChan chan []*message.Message) {

}
