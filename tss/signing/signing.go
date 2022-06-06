package signing

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ChainSafe/chainbridge-core/comm"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/binance-chain/tss-lib/ecdsa/signing"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

var (
	SigningTimeout = time.Minute * 15
)

type SaveDataFetcher interface {
	GetKeyshare() (store.Keyshare, error)
	LockKeyshare()
	UnlockKeyshare()
}

type Signing struct {
	common.BaseTss
	coordinator    bool
	key            store.Keyshare
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
			Cancel:        func() {},
		},
		key: key,
		msg: msg,
	}, nil
}

// Start initializes the signing party and starts the signing tss process.
// Params contains peer subset that leaders sends with start message.
func (s *Signing) Start(
	ctx context.Context,
	coordinator bool,
	resultChn chan interface{},
	errChn chan error,
	params []byte,
) {
	s.coordinator = coordinator
	s.ErrChn = errChn
	s.resultChn = resultChn
	ctx, s.Cancel = context.WithCancel(ctx)

	peerSubset, err := s.unmarshallStartParams(params)
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
	msgChn := make(chan *comm.WrappedMessage)
	s.subscriptionID = s.Communication.Subscribe(s.SessionID(), comm.TssKeySignMsg, msgChn)
	go s.ProcessOutboundMessages(ctx, outChn, comm.TssKeySignMsg)
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

	sortedPeers := common.SortPeersForSession(peers, s.SessionID())
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
func (s *Signing) processEndMessage(ctx context.Context, endChn chan *signing.SignatureData) {
	ticker := time.NewTicker(s.Timeout)
	for {
		select {
		case sig := <-endChn:
			{
				s.Log.Info().Msg("Successfully generated signature")

				if s.coordinator {
					s.resultChn <- sig
				}
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
