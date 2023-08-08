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

	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/connection"
	"github.com/binance-chain/tss-lib/common"
	"github.com/sourcegraph/conc/pool"

	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/author"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	ethCommon "github.com/ethereum/go-ethereum/common"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"

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
	HandleMessage(m *message.Message) (*chains.Proposal, error)
}

type BridgePallet interface {
	IsProposalExecuted(p *chains.Proposal) (bool, error)
	ExecuteProposals(proposals []*chains.Proposal, signature []byte) (types.Hash, *author.ExtrinsicStatusSubscription, error)
	ProposalsHash(proposals []*chains.Proposal) ([]byte, error)
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
func (e *Executor) Execute(ctx context.Context, msgs []*message.Message) error {
	e.exitLock.RLock()
	defer e.exitLock.RUnlock()
	spancContext, span := otel.Tracer("relayer-sygma").Start(ctx, "relayer.sygma.substrate.Execute")
	defer span.End()
	logger := log.With().Str("dd.trace_id", span.SpanContext().TraceID().String()).Logger()

	proposals := make([]*chains.Proposal, 0)
	for _, m := range msgs {
		logger.Info().Str("msg.id", m.ID()).Msgf("Message to execute %s", m.String())
		span.AddEvent("Message to execute received", traceapi.WithAttributes(attribute.String("msg.id", m.ID()), attribute.String("msg.type", string(m.Type))))
		prop, err := e.mh.HandleMessage(m)
		if err != nil {
			return fmt.Errorf("failed to handle message %s with error: %w", m.String(), err)
		}

		isExecuted, err := e.bridge.IsProposalExecuted(prop)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		if isExecuted {
			logger.Info().Str("msg.id", m.ID()).Msgf("Message already executed %s", m.String())
			span.AddEvent("Message already executed", traceapi.WithAttributes(attribute.String("msg.id", m.ID()), attribute.String("msg.type", string(m.Type))))
			continue
		}

		proposals = append(proposals, prop)
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
		span.SpanContext().TraceID().String(),
		e.host,
		e.comm,
		e.fetcher)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	sigChn := make(chan interface{})
	executionContext, cancelExecution := context.WithCancel(spancContext)
	watchContext, cancelWatch := context.WithCancel(spancContext)

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

func (e *Executor) watchExecution(ctx context.Context, cancelExecution context.CancelFunc, proposals []*chains.Proposal, sigChn chan interface{}, sessionID string) error {
	ctxWithSpan, span := otel.Tracer("relayer-sygma").Start(ctx, "relayer.sygma.substrate.watchExecution")
	defer span.End()
	logger := log.With().Str("dd.trace_id", span.SpanContext().TraceID().String()).Logger()
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
				hash, sub, err := e.executeProposal(ctxWithSpan, proposals, signatureData)
				if err != nil {
					_ = e.comm.Broadcast(ctxWithSpan, e.host.Peerstore().Peers(), []byte{}, comm.TssFailMsg, sessionID)
					span.SetStatus(codes.Error, fmt.Errorf("executing proposel has failed %w", err).Error())
					return err
				}

				return e.bridge.TrackExtrinsic(hash, sub)
			}
		case <-ticker.C:
			{
				if !e.areProposalsExecuted(proposals, sessionID) {
					continue
				}

				logger.Info().Str("SessionID", sessionID).Msgf("Successfully executed proposals")
				span.AddEvent("Proposals executed", traceapi.WithAttributes(attribute.String("session.id", sessionID)))
				return nil
			}
		case <-timeout.C:
			{
				span.SetStatus(codes.Error, fmt.Errorf("execution timed out in %s", signingTimeout).Error())
				return fmt.Errorf("execution timed out in %s", signingTimeout)
			}
		case <-ctx.Done():
			{
				return nil
			}
		}
	}
}

func (e *Executor) executeProposal(ctx context.Context, proposals []*chains.Proposal, signatureData *common.SignatureData) (types.Hash, *author.ExtrinsicStatusSubscription, error) {
	_, span := otel.Tracer("relayer-sygma").Start(ctx, "relayer.sygma.substrate.executeProposal")
	defer span.End()
	sig := []byte{}
	sig = append(sig[:], ethCommon.LeftPadBytes(signatureData.R, 32)...)
	sig = append(sig[:], ethCommon.LeftPadBytes(signatureData.S, 32)...)
	sig = append(sig[:], signatureData.SignatureRecovery...)
	sig[len(sig)-1] += 27 // Transform V from 0/1 to 27/28

	hash, sub, err := e.bridge.ExecuteProposals(proposals, sig)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return types.Hash{}, nil, err
	}

	span.AddEvent("Deposit executed", traceapi.WithAttributes(attribute.String("tx.hash", hash.Hex())))
	return hash, sub, err
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
