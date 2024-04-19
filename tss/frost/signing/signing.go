// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package signing

import (
	"context"
	"encoding/json"
	"math/big"

	errors "github.com/ChainSafe/sygma-relayer/tss"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sourcegraph/conc/pool"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
	"golang.org/x/exp/slices"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/keyshare"
	"github.com/ChainSafe/sygma-relayer/tss/frost/common"
	"github.com/ChainSafe/sygma-relayer/tss/util"
)

type SaveDataFetcher interface {
	GetKeyshare() (keyshare.FrostKeyshare, error)
	LockKeyshare()
	UnlockKeyshare()
}

type Signing struct {
	common.BaseFrostTss
	coordinator    bool
	key            keyshare.FrostKeyshare
	msg            *big.Int
	resultChn      chan interface{}
	subscriptionID comm.SubscriptionID
}

func NewSigning(
	msg *big.Int,
	sessionID string,
	host host.Host,
	comm comm.Communication,
	fetcher SaveDataFetcher,
) (*Signing, error) {
	fetcher.LockKeyshare()
	defer fetcher.UnlockKeyshare()
	key, err := fetcher.GetKeyshare()
	if err != nil {
		return nil, err
	}

	return &Signing{
		BaseFrostTss: common.BaseFrostTss{
			Host:          host,
			Communication: comm,
			Peers:         key.Peers,
			SID:           sessionID,
			Log:           log.With().Str("SessionID", sessionID).Str("Process", "signing").Logger(),
			Cancel:        func() {},
		},
		key: key,
		msg: msg,
	}, nil
}

// Run initializes the signing party and runs the signing tss process.
// Params contains peer subset that leaders sends with start message.
func (s *Signing) Run(
	ctx context.Context,
	coordinator bool,
	resultChn chan interface{},
	params []byte,
) error {
	s.coordinator = coordinator
	s.resultChn = resultChn
	ctx, s.Cancel = context.WithCancel(ctx)

	peerSubset, err := s.unmarshallStartParams(params)
	if err != nil {
		return err
	}
	s.Peers = peerSubset

	if IsParticipant(s.Host.ID(), peerSubset) {
		return &errors.SubsetError{Peer: s.Host.ID()}
	}

	s.Handler, err = protocol.NewMultiHandler(
		frost.SignTaproot(
			s.key.Key,
			common.PartyIDSFromPeers(append(s.Host.Peerstore().Peers(), s.Host.ID())),
			s.msg.Bytes(),
		),
		[]byte(s.SessionID()))
	if err != nil {
		return err
	}

	outChn := make(chan tss.Message)
	msgChn := make(chan *comm.WrappedMessage)
	s.subscriptionID = s.Communication.Subscribe(s.SessionID(), comm.TssKeySignMsg, msgChn)

	defer s.Stop()
	p := pool.New().WithContext(ctx).WithCancelOnError()
	p.Go(func(ctx context.Context) error { return s.ProcessOutboundMessages(ctx, outChn, comm.TssKeySignMsg) })
	p.Go(func(ctx context.Context) error { return s.ProcessInboundMessages(ctx, msgChn) })
	p.Go(func(ctx context.Context) error { return s.processEndMessage(ctx) })

	s.Log.Info().Msgf("Started signing process")
	return p.Wait()
}

// Stop ends all subscriptions created when starting the tss process.
func (s *Signing) Stop() {
	log.Info().Str("sessionID", s.SessionID()).Msgf("Stopping tss process.")
	s.Communication.UnSubscribe(s.subscriptionID)
	s.Cancel()
}

// Ready returns true if threshold+1 parties are ready to start the signing process.
func (s *Signing) Ready(readyMap map[peer.ID]bool, excludedPeers []peer.ID) (bool, error) {
	readyMap = s.readyParticipants(readyMap)
	return len(readyMap) == s.key.Threshold+1, nil
}

// ValidCoordinators returns only peers that have a valid keyshare
func (s *Signing) ValidCoordinators() []peer.ID {
	return s.key.Peers
}

// StartParams returns peer subset for this tss process. It is calculated
// by sorting hashes of peer IDs and session ID and chosing ready peers alphabetically
// until threshold is satisfied.
func (s *Signing) StartParams(readyMap map[peer.ID]bool) []byte {
	readyMap = s.readyParticipants(readyMap)
	peers := []peer.ID{}
	for peer := range readyMap {
		peers = append(peers, peer)
	}

	sortedPeers := util.SortPeersForSession(peers, s.SessionID())
	peerSubset := []peer.ID{}
	for _, peer := range sortedPeers {
		peerSubset = append(peerSubset, peer.ID)
		if len(peerSubset) == s.key.Threshold+1 {
			break
		}
	}

	paramBytes, _ := json.Marshal(peerSubset)
	return paramBytes
}

func (s *Signing) unmarshallStartParams(paramBytes []byte) ([]peer.ID, error) {
	var peerSubset []peer.ID
	err := json.Unmarshal(paramBytes, &peerSubset)
	if err != nil {
		return []peer.ID{}, err
	}

	return peerSubset, nil
}

// processEndMessage routes signature to result channel.
func (s *Signing) processEndMessage(ctx context.Context) error {
	defer s.Cancel()
	for {
		select {
		case <-s.Done:
			{
				s.Log.Info().Msg("Successfully generated signature")

				result, err := s.Handler.Result()
				if err != nil {
					return err
				}
				signature := result.(frost.Signature)

				if s.coordinator {
					s.resultChn <- signature
				} else {
					s.resultChn <- nil
				}

				return nil
			}
		case <-ctx.Done():
			{
				return nil
			}
		}
	}
}

// readyParticipants returns all ready peers that contain a valid key share
func (s *Signing) readyParticipants(readyMap map[peer.ID]bool) map[peer.ID]bool {
	readyParticipants := make(map[peer.ID]bool)
	for peer, ready := range readyMap {
		if !ready {
			continue
		}

		if !slices.Contains(s.key.Peers, peer) {
			continue
		}

		readyParticipants[peer] = true
	}

	return readyParticipants
}

func (s *Signing) Retryable() bool {
	return true
}
