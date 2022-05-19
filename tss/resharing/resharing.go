package resharing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ChainSafe/chainbridge-core/communication"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/ecdsa/resharing"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
)

var (
	SigningTimeout = time.Minute * 15
)

type startParams struct {
	OldThreshold int       `json:"oldThreshold"`
	OldSubset    []peer.ID `json:"oldSubset"`
}

type SaveDataStorer interface {
	GetKeyshare() (store.Keyshare, error)
	StoreKeyshare(keyshare store.Keyshare) error
	LockKeyshare()
	UnlockKeyshare()
}

type Resharing struct {
	common.BaseTss
	key            store.Keyshare
	subscriptionID communication.SubscriptionID
	storer         SaveDataStorer
	newThreshold   int
}

func NewResharing(
	sessionID string,
	threshold int,
	host host.Host,
	comm communication.Communication,
	storer SaveDataStorer,
) *Resharing {
	storer.LockKeyshare()
	var key store.Keyshare
	key, err := storer.GetKeyshare()
	if err != nil {
		// empty key for parties that don't have one
		key = store.Keyshare{}
	}

	partyStore := make(map[string]*tss.PartyID)
	return &Resharing{
		BaseTss: common.BaseTss{
			PartyStore:    partyStore,
			Host:          host,
			Communication: comm,
			Peers:         host.Peerstore().Peers(),
			SID:           sessionID,
			Log:           log.With().Str("SessionID", sessionID).Str("Process", "resharing").Logger(),
			Timeout:       SigningTimeout,
			Cancel:        func() {},
		},
		key:          key,
		storer:       storer,
		newThreshold: threshold,
	}
}

// Start initializes the signing party and starts the signing tss procesr.
// Params contains peer subset that leaders sends with start message.
func (r *Resharing) Start(
	ctx context.Context,
	coordinator bool,
	resultChn chan interface{},
	errChn chan error,
	params []byte,
) {
	r.ErrChn = errChn
	ctx, r.Cancel = context.WithCancel(ctx)

	startParams, err := r.unmarshallStartParams(params)
	if err != nil {
		r.ErrChn <- err
		return
	}

	oldParties := common.PartiesFromPeers(startParams.OldSubset)
	oldCtx := tss.NewPeerContext(oldParties)
	newParties := r.sortParties(common.PartiesFromPeers(r.Host.Peerstore().Peers()), oldParties)
	newCtx := tss.NewPeerContext(newParties)
	r.PopulatePartyStore(newParties)
	tssParams := tss.NewReSharingParameters(
		oldCtx,
		newCtx,
		r.PartyStore[r.Host.ID().Pretty()],
		len(oldParties),
		startParams.OldThreshold,
		len(newParties),
		r.newThreshold,
	)
	endChn := make(chan keygen.LocalPartySaveData)
	outChn := make(chan tss.Message)
	msgChn := make(chan *communication.WrappedMessage)
	r.subscriptionID = r.Communication.Subscribe(r.SessionID(), communication.TssReshareMsg, msgChn)
	go r.ProcessOutboundMessages(ctx, outChn, communication.TssReshareMsg)
	go r.ProcessInboundMessages(ctx, msgChn)
	go r.processEndMessage(ctx, endChn)

	r.Log.Info().Msgf("Started resharing process")
	r.Party = resharing.NewLocalParty(tssParams, r.key.Key, outChn, endChn)
	go func() {
		err := r.Party.Start()
		if err != nil {
			r.ErrChn <- err
		}
	}()
}

// Stop ends all subscriptions created when starting the tss process and unlocks keyshare.
func (r *Resharing) Stop() {
	log.Info().Str("sessionID", r.SessionID()).Msgf("Stopping tss process.")
	r.Communication.UnSubscribe(r.subscriptionID)
	r.storer.UnlockKeyshare()
	r.Cancel()
}

// Ready returns true if all parties from peerstore are ready
func (r *Resharing) Ready(readyMap map[peer.ID]bool, excludedPeers []peer.ID) (bool, error) {
	return len(readyMap) == len(r.Host.Peerstore().Peers()), nil
}

// ValidCoordinators returns only peers that have a valid keyshare from the previous resharing
func (r *Resharing) ValidCoordinators() []peer.ID {
	return r.key.Peers
}

// StartParams returns threshold and peer subset from the old key to share with new parties.
func (r *Resharing) StartParams(readyMap map[peer.ID]bool) []byte {
	startParams := &startParams{
		OldThreshold: r.key.Threshold,
		OldSubset:    r.key.Peers,
	}
	paramBytes, _ := json.Marshal(startParams)
	return paramBytes
}

func (r *Resharing) unmarshallStartParams(paramBytes []byte) (startParams, error) {
	var startParams startParams
	err := json.Unmarshal(paramBytes, &startParams)
	if err != nil {
		return startParams, err
	}

	return startParams, nil
}

// processEndMessage routes signature to result channel.
func (r *Resharing) processEndMessage(ctx context.Context, endChn chan keygen.LocalPartySaveData) {
	ticker := time.NewTicker(r.Timeout)
	for {
		select {
		case key := <-endChn:
			{
				r.Log.Info().Msg("Successfully reshared key")

				keyshare := store.NewKeyshare(key, r.newThreshold, r.Peers)
				err := r.storer.StoreKeyshare(keyshare)
				r.ErrChn <- err
				return
			}
		case <-ticker.C:
			{
				r.ErrChn <- fmt.Errorf("reshare process timed out in: %s", SigningTimeout)
				return
			}
		case <-ctx.Done():
			{
				return
			}
		}
	}
}

// sortParties assign new parties indexes that are greater than old party indexes to prevent
// errors when assigning message to a party
func (r *Resharing) sortParties(parties tss.SortedPartyIDs, oldParties tss.SortedPartyIDs) tss.SortedPartyIDs {
	newParties := make(tss.SortedPartyIDs, len(parties))
	copy(newParties, oldParties)
	index := len(oldParties)
	for _, party := range parties {
		if !common.IsParticipant(party, oldParties) {
			newParties[index] = party
			newParties[index].Index = index
			index++
		}
	}
	return newParties
}
