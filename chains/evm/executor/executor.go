// Copyright 2021 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"context"
	"fmt"
	"math/big"
	"time"

	tssSigning "github.com/binance-chain/tss-lib/ecdsa/signing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/rs/zerolog/log"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/executor/proposal"
	"github.com/ChainSafe/chainbridge-core/relayer/message"

	"github.com/ChainSafe/chainbridge-hub/comm"
	"github.com/ChainSafe/chainbridge-hub/tss"
	"github.com/ChainSafe/chainbridge-hub/tss/signing"
)

var (
	executionCheckPeriod = time.Second * 15
)

type MessageHandler interface {
	HandleMessage(m *message.Message) (*proposal.Proposal, error)
}

type BridgeContract interface {
	IsProposalExecuted(p *proposal.Proposal) (bool, error)
	ExecuteProposal(proposal *proposal.Proposal, signature []byte, opts transactor.TransactOptions) (*common.Hash, error)
	ProposalHash(proposal *proposal.Proposal) ([]byte, error)
}

type Executor struct {
	coordinator *tss.Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher
	bridge      BridgeContract
	mh          MessageHandler
}

func NewExecutor(
	host host.Host,
	comm comm.Communication,
	coordinator *tss.Coordinator,
	mh MessageHandler,
	bridgeContract BridgeContract,
	fetcher signing.SaveDataFetcher,
) *Executor {
	return &Executor{
		host:        host,
		comm:        comm,
		coordinator: coordinator,
		mh:          mh,
		bridge:      bridgeContract,
		fetcher:     fetcher,
	}
}

// Execute starts a signing process and executes proposal when signature is generated
func (e *Executor) Execute(m *message.Message) error {
	prop, err := e.mh.HandleMessage(m)
	if err != nil {
		return err
	}

	isExecuted, err := e.bridge.IsProposalExecuted(prop)
	if err != nil {
		return err
	}
	if isExecuted {
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
		e.sessionID(m.Destination, m.DepositNonce),
		e.host,
		e.comm,
		e.fetcher)
	if err != nil {
		return err
	}

	sigChn := make(chan interface{})
	statusChn := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go e.coordinator.Execute(ctx, signing, sigChn, statusChn)

	ticker := time.NewTicker(executionCheckPeriod)
	defer ticker.Stop()
	defer cancel()
	for {
		select {
		case sigResult := <-sigChn:
			{
				signatureData := sigResult.(*tssSigning.SignatureData)
				hash, err := e.executeProposal(prop, signatureData)
				if err != nil {
					return err
				}

				log.Info().Msgf("Sent proposal %v execution with hash: %s", prop, hash)
			}
		case <-ticker.C:
			{
				isExecuted, err := e.bridge.IsProposalExecuted(prop)
				if err != nil || !isExecuted {
					continue
				}

				log.Info().Msgf("Successfully executed proposal %v", prop)
				return nil
			}
		}
	}
}

func (e *Executor) executeProposal(prop *proposal.Proposal, signatureData *tssSigning.SignatureData) (*common.Hash, error) {
	sig := signatureData.Signature.R
	sig = append(sig[:], signatureData.Signature.S[:]...)
	sig = append(sig[:], signatureData.Signature.SignatureRecovery...)
	sig[64] += 27 // Transform V from 0/1 to 27/28

	hash, err := e.bridge.ExecuteProposal(prop, sig, transactor.TransactOptions{
		Priority: prop.Metadata.Priority,
	})
	if err != nil {
		return nil, err
	}

	return hash, err
}

func (e *Executor) sessionID(destination uint8, depositNonce uint64) string {
	return fmt.Sprintf("%d-%d", destination, depositNonce)
}
