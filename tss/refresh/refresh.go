package refresh

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/binance-chain/tss-lib/ecdsa/signing"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
)

var (
	SigningTimeout = time.Minute * 15
)

type SaveDataFetcher interface {
	GetKeyshare() (store.Keyshare, error)
	LockKeyshare()
	UnlockKeyshare()
}

type Refresh struct {
	common.BaseTss
	coordinator    bool
	key            store.Keyshare
	msg            *big.Int
	resultChn      chan interface{}
	subscriptionID communication.SubscriptionID
}

func NewRefresh(
	sessionID string,
	host host.Host,
	comm communication.Communication,
	fetcher SaveDataFetcher,
) (*Refresh, error) {
	key, err := fetcher.GetKeyshare()
	if err != nil {
		return nil, err
	}

	partyStore := make(map[string]*tsr.PartyID)
	return &Refresh{
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

// Start initializes the signing party and starts the signing tss procesr.
// Params contains peer subset that leaders sends with start message.
func (r *Refresh) Start(
	ctx context.Context,
	coordinator bool,
	resultChn chan interface{},
	errChn chan error,
	params []string,
) {
	r.coordinator = coordinator
	r.ErrChn = errChn
	r.resultChn = resultChn
	ctx, r.Cancel = context.WithCancel(ctx)

	peerSubset, err := common.PeersFromIDS(params)
	if err != nil {
		r.ErrChn <- err
		return
	}

	if !common.IsParticipant(common.CreatePartyID(r.Host.ID().Pretty()), common.PartiesFromPeers(peerSubset)) {
		r.Log.Info().Msgf("Party is not in signing subset")
		r.ErrChn <- nil
		return
	}

	r.Peers = peerSubset
	parties := common.PartiesFromPeers(r.Peers)
	r.PopulatePartyStore(parties)
	pCtx := tsr.NewPeerContext(parties)
	tssParams := tsr.NewParameters(pCtx, r.PartyStore[r.Host.ID().Pretty()], len(parties), r.key.Threshold)

	sigChn := make(chan *signing.SignatureData)
	outChn := make(chan tsr.Message)
	msgChn := make(chan *communication.WrappedMessage)
	r.subscriptionID = r.Communication.Subscribe(r.SessionID(), communication.TssKeySignMsg, msgChn)
	go r.ProcessOutboundMessages(ctx, outChn, communication.TssKeySignMsg)
	go r.ProcessInboundMessages(ctx, msgChn)
	go r.processEndMessage(ctx, sigChn)

	r.Log.Info().Msgf("Started signing process")

	r.Party = signing.NewLocalParty(r.msg, tssParams, r.key.Key, outChn, sigChn)
	go func() {
		err := r.Party.Start()
		if err != nil {
			r.ErrChn <- err
		}
	}()
}

// Stop ends all subscriptions created when starting the tss process and unlocks keyshare.
func (r *Refresh) Stop() {
	log.Info().Str("sessionID", r.SessionID()).Msgf("Stopping tss procesr.")
	r.Communication.UnSubscribe(r.subscriptionID)
	r.Cancel()
}

// Ready returns true if threshold+1 parties are ready to start the signing procesr.
func (r *Refresh) Ready(readyMap map[peer.ID]bool, excludedPeers []peer.ID) (bool, error) {
	readyMap = r.readyParticipants(readyMap)
	return len(readyMap) == r.key.Threshold+1, nil
}

// StartParams returns peer subset for this tss procesr. It is calculated
// by sorting hashes of peer IDs and session ID and chosing ready peers alphabetically
// until threshold is satisfied.
func (r *Refresh) StartParams(readyMap map[peer.ID]bool) []string {
	readyMap = r.readyParticipants(readyMap)
	peers := []peer.ID{}
	for peer := range readyMap {
		peers = append(peers, peer)
	}

	sortedPeers := common.SortPeersForSession(peers, r.SessionID())
	params := []string{}
	for _, peer := range sortedPeers {
		params = append(params, peer.ID.Pretty())
		if len(params) == r.key.Threshold+1 {
			break
		}
	}

	return params
}

// processEndMessage routes signature to result channel.
func (r *Refresh) processEndMessage(ctx context.Context, endChn chan *signing.SignatureData) {
	ticker := time.NewTicker(r.Timeout)
	for {
		select {
		case sig := <-endChn:
			{
				r.Log.Info().Msg("Successfully generated signature")

				if r.coordinator {
					r.resultChn <- sig
				}
				r.ErrChn <- nil
				return
			}
		case <-ticker.C:
			{
				r.ErrChn <- fmt.Errorf("signing process timed out in: %s", SigningTimeout)
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
func (r *Refresh) readyParticipants(readyMap map[peer.ID]bool) map[peer.ID]bool {
	readyParticipants := make(map[peer.ID]bool)
	for peer, ready := range readyMap {
		if !ready {
			continue
		}

		if !slicer.Contains(r.key.Peers, peer) {
			continue
		}

		readyParticipants[peer] = true
	}

	return readyParticipants
}
