package signing

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/binance-chain/tss-lib/ecdsa/signing"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
)

var (
	SigningTimeout = time.Minute * 15
)

type SaveDataFetcher interface {
	GetKeyshare() (store.Keyshare, error)
}

type Signing struct {
	common.BaseTss
	key            store.Keyshare
	msg            *big.Int
	resultChn      chan interface{}
	subscriptionID communication.SubscriptionID
}

func NewSigning(
	msg *big.Int,
	sessionID string,
	host host.Host,
	comm communication.Communication,
	fetcher SaveDataFetcher,
) (*Signing, error) {
	key, err := fetcher.GetKeyshare()
	if err != nil {
		return nil, err
	}

	partyStore := make(map[string]*tss.PartyID)
	return &Signing{
		BaseTss: common.BaseTss{
			PartyStore:    partyStore,
			Host:          host,
			Communication: comm,
			Peers:         key.Peers,
			SID:           sessionID,
			Log:           log.With().Str("SessionID", sessionID).Str("Process", "signing").Logger(),
			Timeout:       SigningTimeout,
		},
		key: key,
		msg: msg,
	}, nil
}

// Start initializes the signing party and starts the signing tss process.
// Params contains peer subset that leaders sends with start message.
func (s *Signing) Start(
	ctx context.Context,
	resultChn chan interface{},
	errChn chan error,
	params []string,
) {
	s.ErrChn = errChn
	s.resultChn = resultChn
	ctx, s.Cancel = context.WithCancel(ctx)

	peerSubset, err := common.PeersFromIDS(params)
	if err != nil {
		s.ErrChn <- err
		return
	}

	if !common.IsParticipant(common.CreatePartyID(s.Host.ID().Pretty()), common.PartiesFromPeers(peerSubset)) {
		s.Log.Info().Msgf("Party is not in signing subset")
		s.ErrChn <- nil
		return
	}

	s.Peers = peerSubset
	parties := common.PartiesFromPeers(s.Peers)
	s.PopulatePartyStore(parties)
	pCtx := tss.NewPeerContext(parties)
	tssParams := tss.NewParameters(pCtx, s.PartyStore[s.Host.ID().Pretty()], len(parties), s.key.Threshold)

	sigChn := make(chan *signing.SignatureData)
	outChn := make(chan tss.Message)
	msgChn := make(chan *communication.WrappedMessage)
	s.subscriptionID = s.Communication.Subscribe(s.SessionID(), communication.TssKeySignMsg, msgChn)
	go s.ProcessOutboundMessages(ctx, outChn, communication.TssKeySignMsg)
	go s.ProcessInboundMessages(ctx, msgChn)
	go s.processEndMessage(ctx, sigChn)

	s.Log.Info().Msgf("Started signing process")

	s.Party = signing.NewLocalParty(s.msg, tssParams, s.key.Key, outChn, sigChn)
	go func() {
		err := s.Party.Start()
		if err != nil {
			s.ErrChn <- err
		}
	}()
}

// Stop ends all subscriptions created when starting the tss process and unlocks keyshare.
func (s *Signing) Stop() {
	s.Communication.UnSubscribe(s.subscriptionID)
	s.Cancel()
}

// Ready returns true if threshold+1 parties are ready to start the signing process.
func (s *Signing) Ready(readyMap map[peer.ID]bool) bool {
	return len(readyMap) == s.key.Threshold+1
}

// StartParams returns peer subset for this tss process. It is calculated
// by sorting hashes of peer IDs and session ID and chosing ready peers alphabetically
// until threshold is satisfied.
func (s *Signing) StartParams(readyMap map[peer.ID]bool) []string {
	peers := []peer.ID{}
	for peer := range readyMap {
		peers = append(peers, peer)
	}

	sortedPeers := common.SortPeersForSession(peers, s.SessionID())
	params := []string{}
	for _, peer := range sortedPeers {
		params = append(params, peer.ID.Pretty())
		if len(params) == s.key.Threshold+1 {
			break
		}
	}

	return params
}

// processEndMessage routes signature to result channel.
func (s *Signing) processEndMessage(ctx context.Context, endChn chan *signing.SignatureData) {
	ticker := time.NewTicker(s.Timeout)
	for {
		select {
		case sig := <-endChn:
			{
				s.Log.Info().Msg("Successfully generated signature")

				s.resultChn <- sig
				s.ErrChn <- nil
				return
			}
		case <-ticker.C:
			{
				s.ErrChn <- fmt.Errorf("signing process timed out in: %s", SigningTimeout)
				return
			}
		case <-ctx.Done():
			{
				return
			}
		}
	}
}
