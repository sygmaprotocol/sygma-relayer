// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/connection"
	"github.com/binance-chain/tss-lib/common"
	"github.com/sourcegraph/conc/pool"

	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/author"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	ethCommon "github.com/ethereum/go-ethereum/common"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"

	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"

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

type BridgePallet interface {
	IsProposalExecuted(p *proposal.Proposal) (bool, error)
	ExecuteProposals(proposals []*proposal.Proposal, signature []byte) (types.Hash, *author.ExtrinsicStatusSubscription, error)
	ProposalsHash(proposals []*proposal.Proposal) ([]byte, error)
	TrackExtrinsic(extHash types.Hash, sub *author.ExtrinsicStatusSubscription) error
}

type Executor struct {
	coordinator *tss.Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher
	bridge      BridgePallet
	mh          MessageHandler
	conn        *connection.Connection
	exitLock    *sync.RWMutex
}

func NewExecutor(
	host host.Host,
	comm comm.Communication,
	coordinator *tss.Coordinator,
	mh MessageHandler,
	bridgePallet BridgePallet,
	fetcher signing.SaveDataFetcher,
	conn *connection.Connection,
	exitLock *sync.RWMutex,
) *Executor {
	return &Executor{
		host:        host,
		comm:        comm,
		coordinator: coordinator,
		mh:          mh,
		bridge:      bridgePallet,
		fetcher:     fetcher,
		conn:        conn,
		exitLock:    exitLock,
	}
}

// Execute starts a signing process and executes proposals when signature is generated
func (e *Executor) Execute(msgs []*message.Message) error {
	e.exitLock.RLock()
	defer e.exitLock.RUnlock()

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
	executionContext, cancelExecution := context.WithCancel(context.Background())
	watchContext, cancelWatch := context.WithCancel(context.Background())

	pool := pool.New().WithErrors()
	pool.Go(func() error {
		err := e.coordinator.Execute(executionContext, signing, sigChn)
		if err != nil {
			cancelWatch()
		}

		return err
	})
	pool.Go(func() error {
		return e.watchExecution(watchContext, cancelExecution, proposals, sigChn, sessionID)
	})
	return pool.Wait()
}

func (e *Executor) watchExecution(ctx context.Context, cancelExecution context.CancelFunc, proposals []*proposal.Proposal, sigChn chan interface{}, sessionID string) error {
	ticker := time.NewTicker(executionCheckPeriod)
	timeout := time.NewTicker(signingTimeout)
	defer ticker.Stop()
	defer timeout.Stop()
	defer cancelExecution()

	for {
		select {
		case sigResult := <-sigChn:
			{
				cancelExecution()
				if sigResult == nil {
					continue
				}

				signatureData := sigResult.(*common.SignatureData)
				hash, sub, err := e.executeProposal(proposals, signatureData)
				if err != nil {
					_ = e.comm.Broadcast(e.host.Peerstore().Peers(), []byte{}, comm.TssFailMsg, sessionID)
					return err
				}

				return e.bridge.TrackExtrinsic(hash, sub)
			}
		case <-ticker.C:
			{
				if !e.areProposalsExecuted(proposals, sessionID) {
					continue
				}

				log.Info().Str("SessionID", sessionID).Msgf("Successfully executed proposals")
				return nil
			}
		case <-timeout.C:
			{
				return fmt.Errorf("execution timed out in %s", signingTimeout)
			}
		case <-ctx.Done():
			{
				return nil
			}
		}
	}
}

func (e *Executor) executeProposal(proposals []*proposal.Proposal, signatureData *common.SignatureData) (types.Hash, *author.ExtrinsicStatusSubscription, error) {
	sig := []byte{}
	sig = append(sig[:], ethCommon.LeftPadBytes(signatureData.R, 32)...)
	sig = append(sig[:], ethCommon.LeftPadBytes(signatureData.S, 32)...)
	sig = append(sig[:], signatureData.SignatureRecovery...)
	sig[len(sig)-1] += 27 // Transform V from 0/1 to 27/28

	hash, sub, err := e.bridge.ExecuteProposals(proposals, sig)
	if err != nil {
		return types.Hash{}, nil, err
	}

	return hash, sub, err
}

func (e *Executor) areProposalsExecuted(proposals []*proposal.Proposal, sessionID string) bool {
	for _, prop := range proposals {
		isExecuted, err := e.bridge.IsProposalExecuted(prop)
		if err != nil || !isExecuted {
			return false
		}
	}

	return true
}

func (e *Executor) sessionID(hash []byte) string {
	return fmt.Sprintf("signing-%s", ethCommon.Bytes2Hex(hash))
}
