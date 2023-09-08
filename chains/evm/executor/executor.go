// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"context"
	"fmt"
	"github.com/ChainSafe/chainbridge-core/observability"
	"math/big"
	"sync"
	"time"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/executor/proposal"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/tss"
	"github.com/ChainSafe/sygma-relayer/tss/signing"
	"github.com/binance-chain/tss-lib/common"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/sourcegraph/conc/pool"
	"go.opentelemetry.io/otel/attribute"
)

var (
	executionCheckPeriod = time.Minute
	signingTimeout       = 30 * time.Minute
)

type MessageHandler interface {
	HandleMessage(m *message.Message) (*proposal.Proposal, error)
}

type BridgeContract interface {
	IsProposalExecuted(p *chains.Proposal) (bool, error)
	ExecuteProposals(ctx context.Context, proposals []*chains.Proposal, signature []byte, opts transactor.TransactOptions) (*ethCommon.Hash, error)
	ProposalsHash(proposals []*chains.Proposal) ([]byte, error)
}

type Executor struct {
	coordinator *tss.Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher
	bridge      BridgeContract
	mh          MessageHandler
	exitLock    *sync.RWMutex
}

func NewExecutor(
	host host.Host,
	comm comm.Communication,
	coordinator *tss.Coordinator,
	mh MessageHandler,
	bridgeContract BridgeContract,
	fetcher signing.SaveDataFetcher,
	exitLock *sync.RWMutex,
) *Executor {
	return &Executor{
		host:        host,
		comm:        comm,
		coordinator: coordinator,
		mh:          mh,
		bridge:      bridgeContract,
		fetcher:     fetcher,
		exitLock:    exitLock,
	}
}

// Execute starts a signing process and executes proposals when signature is generated
func (e *Executor) Execute(ctx context.Context, msgs []*message.Message) error {
	e.exitLock.RLock()
	defer e.exitLock.RUnlock()
	ctx, span, logger := observability.CreateSpanAndLoggerFromContext(ctx, "sygma-relayer", "relayer.sygma.evm.Execute")
	defer span.End()

	proposals := make([]*chains.Proposal, 0)
	for _, m := range msgs {
		observability.LogAndEvent(logger.Debug(), span, "Message to execute received", attribute.String("msg.id", m.ID()), attribute.String("msg.full", m.String()))
		prop, err := e.mh.HandleMessage(m)
		if err != nil {
			return observability.LogAndRecordErrorWithStatus(nil, span, err, "failed to handle message")
		}
		evmProposal := chains.NewProposal(prop.Source, prop.Destination, prop.DepositNonce, prop.ResourceId, prop.Data, prop.Metadata)
		isExecuted, err := e.bridge.IsProposalExecuted(evmProposal)
		if err != nil {
			return observability.LogAndRecordErrorWithStatus(nil, span, err, "failed to call IsProposalExecuted")
		}
		if isExecuted {
			observability.LogAndEvent(logger.Info(), span, "Message already executed")
			continue
		}
		observability.LogAndEvent(logger.Info(), span, "Executing message")
		proposals = append(proposals, evmProposal)
	}
	if len(proposals) == 0 {
		return nil
	}

	propHash, err := e.bridge.ProposalsHash(proposals)
	if err != nil {
		return observability.LogAndRecordErrorWithStatus(nil, span, err, "failed to build ProposalsHash")
	}

	sessionID := e.sessionID(propHash)
	observability.SetAttrsToSpanAnLogger(&logger, span, attribute.String("tss.session.id", sessionID))

	msg := big.NewInt(0)
	msg.SetBytes(propHash)
	signing, err := signing.NewSigning(
		msg,
		e.sessionID(propHash),
		e.host,
		e.comm,
		e.fetcher)
	if err != nil {
		return observability.LogAndRecordErrorWithStatus(nil, span, err, "failed to create NewSigning")
	}

	sigChn := make(chan interface{})
	executionContext, cancelExecution := context.WithCancel(ctx)
	watchContext, cancelWatch := context.WithCancel(ctx)
	pool := pool.New().WithErrors()
	pool.Go(func() error {
		err := e.coordinator.Execute(executionContext, signing, sigChn)
		if err != nil {
			cancelWatch()
		}
		return err
	})
	pool.Go(func() error { return e.watchExecution(watchContext, cancelExecution, proposals, sigChn, sessionID) })
	return pool.Wait()
}

func (e *Executor) watchExecution(ctx context.Context, cancelExecution context.CancelFunc, proposals []*chains.Proposal, sigChn chan interface{}, sessionID string) error {
	ctx, span, logger := observability.CreateSpanAndLoggerFromContext(ctx, "sygma-relayer", "relayer.sygma.evm.watchExecution")
	defer span.End()
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
				hash, err := e.executeProposal(ctx, proposals, signatureData)
				if err != nil {
					_ = e.comm.Broadcast(ctx, e.host.Peerstore().Peers(), []byte{}, comm.TssFailMsg, sessionID)
					return observability.LogAndRecordErrorWithStatus(nil, span, err, "executing proposal has failed")
				}
				observability.LogAndEvent(logger.Info(), span, "Sent proposals execution with", attribute.String("tx.hash", hash.String()), attribute.String("tss.session.id", sessionID))
			}
		case <-ticker.C:
			{
				if !e.areProposalsExecuted(proposals, sessionID) {
					continue
				}

				observability.LogAndEvent(logger.Info(), span, "Proposals executed", attribute.String("tss.session.id", sessionID))
				return nil
			}
		case <-timeout.C:
			{
				return observability.LogAndRecordErrorWithStatus(nil, span, fmt.Errorf("execution timed out in %s", signingTimeout), "failed to watchExecution")
			}
		case <-ctx.Done():
			{
				return nil
			}
		}
	}
}

func (e *Executor) executeProposal(ctx context.Context, proposals []*chains.Proposal, signatureData *common.SignatureData) (*ethCommon.Hash, error) {
	ctx, span, _ := observability.CreateSpanAndLoggerFromContext(ctx, "sygma-relayer", "relayer.sygma.evm.executeProposal")
	defer span.End()
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

	hash, err := e.bridge.ExecuteProposals(ctx, proposals, sig, transactor.TransactOptions{
		GasLimit: gasLimit,
	})
	if err != nil {
		return nil, observability.LogAndRecordErrorWithStatus(nil, span, err, "failed to ExecuteProposals")
	}
	return hash, err
}

func (e *Executor) areProposalsExecuted(proposals []*chains.Proposal, sessionID string) bool {
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
