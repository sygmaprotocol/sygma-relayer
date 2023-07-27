// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	traceapi "go.opentelemetry.io/otel/trace"
	"math/big"
	"sync"
	"time"

	"github.com/binance-chain/tss-lib/common"
	"github.com/sourcegraph/conc/pool"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/executor/proposal"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/sygma-relayer/chains"
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
	IsProposalExecuted(p *chains.Proposal) (bool, error)
	ExecuteProposals(proposals []*chains.Proposal, signature []byte, opts transactor.TransactOptions) (*ethCommon.Hash, error)
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

	tp := otel.GetTracerProvider()
	_, span := tp.Tracer("relayer-execute").Start(ctx, "relayer.sygma.Execute")
	defer span.End()

	proposals := make([]*chains.Proposal, 0)
	for _, m := range msgs {
		log.Info().Str("msg_id", m.ID()).Msgf("Executing message %s", m.String())
		span.AddEvent("Executing message", traceapi.WithAttributes(attribute.String("msg_id", m.ID()), attribute.String("msg_type", string(m.Type))))
		prop, err := e.mh.HandleMessage(m)
		if err != nil {
			return fmt.Errorf("failed to handle message %s with error: %w", m.String(), err)
		}
		evmProposal := chains.NewProposal(prop.Source, prop.Destination, prop.DepositNonce, prop.ResourceId, prop.Data, prop.Metadata)
		isExecuted, err := e.bridge.IsProposalExecuted(evmProposal)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		if isExecuted {
			log.Info().Str("msg_id", m.ID()).Msgf("Prop %p already executed", prop)
			continue
		}

		proposals = append(proposals, evmProposal)
	}
	if len(proposals) == 0 {
		return nil
	}

	propHash, err := e.bridge.ProposalsHash(proposals)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
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
		span.SetStatus(codes.Error, err.Error())
		return err
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
				hash, err := e.executeProposal(proposals, signatureData)
				if err != nil {
					_ = e.comm.Broadcast(e.host.Peerstore().Peers(), []byte{}, comm.TssFailMsg, sessionID)
					return err
				}

				log.Info().Str("SessionID", sessionID).Msgf("Sent proposals execution with hash: %s", hash)
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

func (e *Executor) executeProposal(proposals []*chains.Proposal, signatureData *common.SignatureData) (*ethCommon.Hash, error) {
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
