// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"github.com/ChainSafe/chainbridge-core/types"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/rs/zerolog/log"
)

type ChainConnection interface {
	UpdateMetatdata() error
}

type SystemUpdateEventHandler struct {
	conn ChainConnection
}

func NewSystemUpdateEventHandler(conn ChainConnection) *SystemUpdateEventHandler {
	return &SystemUpdateEventHandler{
		conn: conn,
	}
}

func (eh *SystemUpdateEventHandler) HandleEvents(evts []*Events, msgChan chan []*message.Message) error {
	for _, evt := range evts {
		if len(evt.System_CodeUpdated) > 0 {
			err := eh.conn.UpdateMetatdata()
			if err != nil {
				log.Error().Err(err).Msg("Unable to update Metadata")
				return err
			}
		}
	}

	return nil
}

type DepositHandler interface {
	HandleDeposit(sourceID uint8, destID uint8, nonce uint64, resourceID types.ResourceID, calldata []byte, depositType message.TransferType, handlerResponse []byte) (*message.Message, error)
}

type FungibleTransferEventHandler struct {
	domainID       uint8
	depositHandler DepositHandler
}

func NewFungibleTransferEventHandler(domainID uint8, depositHandler DepositHandler) *FungibleTransferEventHandler {
	return &FungibleTransferEventHandler{
		depositHandler: depositHandler,
		domainID:       domainID,
	}
}

func (eh *FungibleTransferEventHandler) HandleEvents(evts []*Events, msgChan chan []*message.Message) error {
	domainDeposits := make(map[uint8][]*message.Message)

	for _, evt := range evts {
		for _, d := range evt.Deposit {
			func(d Deposit) {
				defer func() {
					if r := recover(); r != nil {
						log.Error().Msgf("panic occured while handling deposit %+v", d)
					}
				}()

				m, err := eh.depositHandler.HandleDeposit(eh.domainID, d.DestinationDomainID, d.DepositNonce, d.ResourceID, d.Data, d.DepositType, d.HandlerResponse)
				if err != nil {
					log.Error().Err(err).Msgf("%v", err)
					return
				}
				domainDeposits[m.Destination] = append(domainDeposits[m.Destination], m)
			}(d)
		}
	}

	for _, deposits := range domainDeposits {
		msgChan <- deposits
	}
	return nil
}
