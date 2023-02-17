// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package listener

import (
	"math/big"

	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/rs/zerolog/log"
)

type SystemUpdateEventHandler struct {
	conn ChainConnection
}

func NewSystemUpdateEventHandler(conn ChainConnection) *SystemUpdateEventHandler {
	return &SystemUpdateEventHandler{
		conn: conn,
	}
}

func (eh *SystemUpdateEventHandler) HandleEvents(evts *events.Events, msgChan chan []*message.Message) error {
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

func (eh *FungibleTransferEventHandler) HandleEvents(evts *events.Events, msgChan chan []*message.Message) error {
	domainDeposits := make(map[uint8][]*message.Message)
	for _, d := range evts.SygmaBridge_Deposit {
		func(d events.EventDeposit) {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msgf("panic occured while handling deposit %+v", d)
				}
			}()

			m, err := eh.depositHandler.HandleDeposit(eh.domainID, d.DestDomainID, d.DepositNonce, d.ResourceID, d.CallData, d.TransferType)
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

type RetryEventHandler struct {
	conn               ChainConnection
	domainID           uint8
	blockConfirmations *big.Int
	depositHandler     DepositHandler
}

func NewRetryEventHandler(conn ChainConnection, depositHandler DepositHandler, domainID uint8, blockConfirmations *big.Int) *RetryEventHandler {
	return &RetryEventHandler{
		depositHandler:     depositHandler,
		domainID:           domainID,
		blockConfirmations: blockConfirmations,
		conn:               conn,
	}
}

func (rh *RetryEventHandler) HandleEvents(evts []*events.Events, msgChan chan []*message.Message) error {
	latest, err := rh.conn.GetBlockLatest()
	if err != nil {
		return err
	}
	latestBlockNumber := big.NewInt(int64(latest.Block.Header.Number))

	domainDeposits := make(map[uint8][]*message.Message)
	for _, evt := range evts {
		for _, r := range evt.SygmaBridge_Retry {
			err := func(er events.EventRetry) error {
				defer func() {
					if r := recover(); r != nil {
						log.Error().Msgf("panic occured while handling retry event %+v", evt)
					}
				}()

				if new(big.Int).Sub(latestBlockNumber, er.DepositOnBlockHeight.Int).Cmp(big.NewInt(rh.blockConfirmations.Int64())) == -1 {
					log.Warn().Msgf("Retry event for block number %d has not enough confirmations", er.DepositOnBlockHeight)
					return nil
				}

				bh, err := rh.conn.GetBlockHash(er.DepositOnBlockHeight.Uint64())
				if err != nil {
					return err
				}

				bEvts, err := rh.conn.GetBlockEvents(bh)
				if err != nil {
					return err
				}

				for _, d := range bEvts.SygmaBridge_Deposit {
					m, err := rh.depositHandler.HandleDeposit(rh.domainID, d.DestDomainID, d.DepositNonce, d.ResourceID, d.CallData, d.TransferType)
					if err != nil {
						return err
					}

					domainDeposits[m.Destination] = append(domainDeposits[m.Destination], m)
				}

				return nil
			}(r)
			if err != nil {
				return err
			}
		}
	}

	for _, deposits := range domainDeposits {
		msgChan <- deposits
	}
	return nil
}
