package signing

import (
	"context"
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
	LockKeyshare()
	UnlockKeyshare()
}

type Signing struct {
	common.BaseTss
	key            store.Keyshare
	msg            *big.Int
	sigChn         chan *signing.SignatureData
	subscriptionID string
}

func NewSigning(
	msg *big.Int,
	sessionID string,
	host host.Host,
	comm communication.Communication,
	fetcher SaveDataFetcher,
	errChn chan error,
	sigChn chan *signing.SignatureData,
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
			ErrChn:        errChn,
			Timeout:       SigningTimeout,
		},
		key:    key,
		msg:    msg,
		sigChn: sigChn,
	}, nil
}

// Start initializes the signing party and starts the signing tss process.
// Params contains peer subset that leaders sends with start message.
func (s *Signing) Start(ctx context.Context, params []string) {
	peerSubset, err := common.PeersFromIDS(params)
	if err != nil {
		s.ErrChn <- err
		return
	}

	s.Peers = peerSubset
	parties := common.PartiesFromPeers(s.Peers)
	s.PopulatePartyStore(parties)
	pCtx := tss.NewPeerContext(parties)
	tssParams := tss.NewParameters(pCtx, s.PartyStore[s.Host.ID().Pretty()], len(parties), s.key.Threshold)

	outChn := make(chan tss.Message)
	msgChn := make(chan *communication.WrappedMessage)
	s.subscriptionID = s.Communication.Subscribe(communication.TssKeySignMsg, s.SessionID(), msgChn)
	go s.ProcessOutboundMessages(ctx, outChn, communication.TssKeySignMsg)
	go s.ProcessInboundMessages(ctx, msgChn)

	s.Party = signing.NewLocalParty(big.NewInt(0), tssParams, s.key.Key, outChn, s.sigChn)
	go func() {
		err := s.Party.Start()
		if err != nil {
			s.ErrChn <- err
		}
	}()

	s.Log.Info().Msgf("Started signing process")
}

// Stop ends all subscriptions created when starting the tss process and unlocks keyshare.
func (s *Signing) Stop() {
	s.Communication.UnSubscribe(s.subscriptionID)
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
	params := make([]string, len(peers))
	for i, peer := range sortedPeers {
		params[i] = peer.ID.Pretty()

		if len(params) == s.key.Threshold+1 {
			break
		}
	}

	return params
}
