// Copyright 2021 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/ChainSafe/chainbridge-core/tss"
	"github.com/ChainSafe/chainbridge-core/tss/signing"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/rs/zerolog/log"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/executor/proposal"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	tssSigning "github.com/binance-chain/tss-lib/ecdsa/signing"
)

type MessageHandler interface {
	HandleMessage(m *message.Message) (*proposal.Proposal, error)
}

type BridgeContract interface {
	ProposalStatus(p *proposal.Proposal) (message.ProposalStatus, error)
	ExecuteProposal(proposal *proposal.Proposal, signature []byte, opts transactor.TransactOptions) (*common.Hash, error)
	ProposalHash(proposal *proposal.Proposal) ([]byte, error)
}

type Executor struct {
	host    host.Host
	comm    communication.Communication
	fetcher signing.SaveDataFetcher
	bridge  BridgeContract
	mh      MessageHandler
}

func NewExecutor(
	host host.Host,
	comm communication.Communication,
	mh MessageHandler,
	bridgeContract BridgeContract,
	fetcher signing.SaveDataFetcher,
) *Executor {
	return &Executor{
		host:    host,
		comm:    comm,
		mh:      mh,
		bridge:  bridgeContract,
		fetcher: fetcher,
	}
}

// Execute starts a signing process and executes proposal when signature is generated
func (e *Executor) Execute(m *message.Message) error {
	prop, err := e.mh.HandleMessage(m)
	if err != nil {
		return err
	}

	ps, err := e.bridge.ProposalStatus(prop)
	if err != nil {
		return err
	}
	if ps.Status == message.ProposalStatusExecuted {
		return nil
	}

	propHash, err := e.bridge.ProposalHash(prop)
	if err != nil {
		return err
	}

	msg := big.NewInt(0)
	msg.SetBytes(propHash)
	signing, err := signing.NewSigning(
		msg,
		fmt.Sprintf("%d-%d", m.Destination, m.DepositNonce),
		e.host,
		e.comm,
		e.fetcher)
	if err != nil {
		return err
	}
	coordinator := tss.NewCoordinator(e.host, signing, e.comm)
	sigChn := make(chan interface{})
	statusChn := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())
	go coordinator.Execute(ctx, sigChn, statusChn)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case sigResult := <-sigChn:
			{
				signatureData := sigResult.(*tssSigning.SignatureData)
				sig := signatureData.Signature.R
				sig = append(sig[:], signatureData.Signature.S[:]...)
				sig = append(sig[:], signatureData.Signature.SignatureRecovery...)
				sig[64] += 27

				sigPublicKey, err := crypto.Ecrecover(propHash, sig)
				if err != nil {
					log.Err(err).Msgf("Failed recovering signature")
				}

				log.Info().Msgf("Signature public key: %s", common.BytesToAddress(sigPublicKey))

				hash, err := e.bridge.ExecuteProposal(prop, sig, transactor.TransactOptions{})
				if err != nil {
					cancel()
					return err
				}

				log.Info().Msgf("Sent proposal %v execution with hash: %s", prop, hash)
			}
		case status := <-statusChn:
			{
				log.Info().Msgf("Excited execution of proposal %v with status: %v", prop, status)

				cancel()
				return status
			}
		case <-ticker.C:
			{
				ps, err := e.bridge.ProposalStatus(prop)
				if err != nil {
					continue
				}
				if ps.Status != message.ProposalStatusExecuted {
					continue
				}

				log.Info().Msgf("Successfully executed proposal %v", prop)
				cancel()
				return nil
			}
		}
	}
}
