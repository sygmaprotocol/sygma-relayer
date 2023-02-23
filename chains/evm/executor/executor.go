// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/binance-chain/tss-lib/common"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/executor/proposal"
	"github.com/ChainSafe/chainbridge-core/relayer/message"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/tss"
	"github.com/ChainSafe/sygma-relayer/tss/signing"
)

var (
	executionCheckPeriod = time.Minute
	signingTimeout       = 30 * time.Minute
)

type MessageHandler interface {
	HandleMessage(m *message.Message) (*proposal.Proposal, error)
}

type BridgeContract interface {
	IsProposalExecuted(p *proposal.Proposal) (bool, error)
	ExecuteProposals(proposals []*proposal.Proposal, signature []byte, opts transactor.TransactOptions) (*ethCommon.Hash, error)
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
	proposals := make([]*proposal.Proposal, 0)
	for _, m := range msgs {
		prop, err := e.mh.HandleMessage(m)
		if err != nil {
			return err
		}

		isExecuted, err := e.bridge.IsProposalExecuted(prop)
		if err != nil {
			return err
		}
		if isExecuted {
			log.Info().Msgf("Prop %p already executed", prop)
			continue
		}

		proposals = append(proposals, prop)
	}
	if len(proposals) == 0 {
		return nil
	}

	propHash, err := e.bridge.ProposalsHash(proposals)
	if err != nil {
		return err
	}

	sessionID := e.sessionID(propHash)
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
	statusChn := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	go e.coordinator.Execute(ctx, signing, sigChn, statusChn)

	ticker := time.NewTicker(executionCheckPeriod)
	timeout := time.NewTicker(signingTimeout)
	defer ticker.Stop()
	defer timeout.Stop()
	defer cancel()
	for {
		select {
		case sigResult := <-sigChn:
			{
				signatureData := sigResult.(*common.SignatureData)
				hash, err := e.executeProposal(proposals, signatureData)
				if err != nil {
					go e.comm.Broadcast(e.host.Peerstore().Peers(), []byte{}, comm.TssFailMsg, sessionID, nil)
					return err
				}

				log.Info().Str("SessionID", sessionID).Msgf("Sent proposals execution with hash: %s", hash)
			}
		case err := <-statusChn:
			{
				return err
			}
		case <-ticker.C:
			{
				allExecuted := true
				for _, prop := range proposals {
					isExecuted, err := e.bridge.IsProposalExecuted(prop)
					if err != nil || !isExecuted {
						allExecuted = false
						continue
					}

					log.Info().Str("SessionID", sessionID).Msgf("Successfully executed proposal %v", prop)
				}

				if allExecuted {
					return nil
				}
			}
		case <-timeout.C:
			{
				return fmt.Errorf("execution timed out in %s", signingTimeout)
			}
		}
	}
}

func (e *Executor) executeProposal(proposals []*proposal.Proposal, signatureData *common.SignatureData) (*ethCommon.Hash, error) {
	sig := []byte{}
	sig = append(sig[:], ethCommon.LeftPadBytes(signatureData.R, 32)...)
	sig = append(sig[:], ethCommon.LeftPadBytes(signatureData.S, 32)...)
	sig = append(sig[:], signatureData.SignatureRecovery...)
	sig[len(sig)-1] += 27 // Transform V from 0/1 to 27/28

	var gasLimit uint64
	l, ok := proposals[0].Metadata.Data["gasLimit"]
	if ok {
		gasLimit = l.(uint64)
	}

	hash, err := e.bridge.ExecuteProposals(proposals, sig, transactor.TransactOptions{
		GasLimit: gasLimit,
	})
	if err != nil {
		return nil, err
	}

	return hash, err
}

func (e *Executor) sessionID(hash []byte) string {
	return fmt.Sprintf("signing-%s", ethCommon.Bytes2Hex(hash))
}
