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

	"github.com/ChainSafe/sygma-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/sygma-core/chains/evm/executor/proposal"
	"github.com/ChainSafe/sygma-core/relayer/message"

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
	ExecuteProposals(proposals []*proposal.Proposal, signature []byte, opts transactor.TransactOptions) (*common.Hash, error)
	ProposalsHash(proposals []*proposal.Proposal) ([]byte, error)
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

// Execute starts a signing process and executes proposals when signature is generated
func (e *Executor) Execute(msgs []*message.Message) error {
	proposals := make([]*proposal.Proposal, len(msgs))
	for i, m := range msgs {
		prop, err := e.mh.HandleMessage(m)
		if err != nil {
			return err
		}

		isExecuted, err := e.bridge.IsProposalExecuted(prop)
		if err != nil {
			return err
		}
		if isExecuted {
			continue
		}

		proposals[i] = prop
	}
	if len(proposals) == 0 {
		return nil
	}

	propHash, err := e.bridge.ProposalsHash(proposals)
	if err != nil {
		return err
	}

	msg := big.NewInt(0)
	msg.SetBytes(propHash)
	signing, err := signing.NewSigning(
		msg,
		e.sessionID(propHash),
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
				hash, err := e.executeProposal(proposals, signatureData)
				if err != nil {
					return err
				}

				log.Info().Msgf("Sent proposals execution with hash: %s", hash)
			}
		case <-ticker.C:
			{
				isExecuted, err := e.bridge.IsProposalExecuted(proposals[0])
				if err != nil || !isExecuted {
					continue
				}

				log.Info().Msgf("Successfully executed proposals %v", proposals)
				return nil
			}
		}
	}
}

func (e *Executor) executeProposal(proposals []*proposal.Proposal, signatureData *tssSigning.SignatureData) (*common.Hash, error) {
	sig := signatureData.Signature.R
	sig = append(sig[:], signatureData.Signature.S[:]...)
	sig = append(sig[:], signatureData.Signature.SignatureRecovery...)
	sig[64] += 27 // Transform V from 0/1 to 27/28

	hash, err := e.bridge.ExecuteProposals(proposals, sig, transactor.TransactOptions{})
	if err != nil {
		return nil, err
	}

	return hash, err
}

func (e *Executor) sessionID(hash []byte) string {
	return fmt.Sprintf("signing-%s", common.Bytes2Hex(hash))
}
