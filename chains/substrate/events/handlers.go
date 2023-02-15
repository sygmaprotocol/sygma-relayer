// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"fmt"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
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

func (eh *SystemUpdateEventHandler) HandleEvents(evts *Events, msgChan chan []*message.Message) error {
	if len(evts.System_CodeUpdated) > 0 {
		err := eh.conn.UpdateMetatdata()
		if err != nil {
			log.Error().Err(err).Msg("Unable to update Metadata")
			return err
		}
	}
	return nil
}

type DepositHandler interface {
	HandleDeposit(sourceID uint8, destID types.U8, nonce types.U64, resourceID types.Bytes32, calldata []byte, transferType [1]byte) (*message.Message, error)
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

func (eh *FungibleTransferEventHandler) HandleEvents(evts *Events, msgChan chan []*message.Message) error {
	fmt.Println("evtsssssssssssssssssssssssss\nqnn\nn")
	fmt.Println(evts)
	domainDeposits := make(map[uint8][]*message.Message)
	for _, d := range evts.SygmaBridge_Deposit {
		func(d EventDeposit) {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msgf("panic occured while handling deposit %+v", d)
				}
			}()

			m, err := eh.depositHandler.HandleDeposit(eh.domainID, d.DestDomainId, d.DepositNonce, d.ResourceID, d.CallData, d.TransferType)
			if err != nil {
				log.Error().Err(err).Msgf("%v", err)
				return
			}
			domainDeposits[m.Destination] = append(domainDeposits[m.Destination], m)
		}(d)

	}
	for _, deposits := range domainDeposits {
		msgChan <- deposits
	}
	return nil
}
