package keygen

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/chainbridge-hub/comm"
	"github.com/ChainSafe/chainbridge-hub/keyshare"
	"github.com/ChainSafe/chainbridge-hub/tss/common"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
)

var (
	KeygenTimeout = time.Minute * 15
)

type SaveDataStorer interface {
	StoreKeyshare(keyshare keyshare.Keyshare) error
	LockKeyshare()
	UnlockKeyshare()
}

type Keygen struct {
	common.BaseTss
	storer         SaveDataStorer
	threshold      int
	subscriptionID comm.SubscriptionID
}

func NewKeygen(
	sessionID string,
	threshold int,
	host host.Host,
	comm comm.Communication,
	storer SaveDataStorer,
) *Keygen {
	partyStore := make(map[string]*tss.PartyID)
	return &Keygen{
		BaseTss: common.BaseTss{
			PartyStore:    partyStore,
			Host:          host,
			Communication: comm,
			Peers:         host.Peerstore().Peers(),
			SID:           sessionID,
			Log:           log.With().Str("SessionID", sessionID).Str("Process", "keygen").Logger(),
			Timeout:       KeygenTimeout,
			Cancel:        func() {},
		},
		storer:    storer,
		threshold: threshold,
	}
}

// Start initializes the keygen party and starts the keygen tss process.
//
// Should be run only after all the participating parties are ready.
func (k *Keygen) Start(
	ctx context.Context,
	coordinator bool,
	resultChn chan interface{},
	errChn chan error,
	params []byte,
) {
	k.ErrChn = errChn
	ctx, k.Cancel = context.WithCancel(ctx)
	k.storer.LockKeyshare()

	parties := common.PartiesFromPeers(k.Host.Peerstore().Peers())
	k.PopulatePartyStore(parties)

	pCtx := tss.NewPeerContext(parties)
	tssParams := tss.NewParameters(pCtx, k.PartyStore[k.Host.ID().Pretty()], len(parties), k.threshold)

	outChn := make(chan tss.Message)
	msgChn := make(chan *comm.WrappedMessage)
	endChn := make(chan keygen.LocalPartySaveData)

	k.subscriptionID = k.Communication.Subscribe(k.SessionID(), comm.TssKeyGenMsg, msgChn)

	go k.ProcessOutboundMessages(ctx, outChn, comm.TssKeyGenMsg)
	go k.ProcessInboundMessages(ctx, msgChn)
	go k.processEndMessage(ctx, endChn)

	k.Party = keygen.NewLocalParty(tssParams, outChn, endChn)

	k.Log.Info().Msgf("Started keygen process")
	go func() {
		err := k.Party.Start()
		if err != nil {
			k.ErrChn <- err
		}
	}()
}

// Stop ends all subscriptions created when starting the tss process and unlocks keyshare.
func (k *Keygen) Stop() {
	k.Communication.UnSubscribe(k.subscriptionID)
	k.storer.UnlockKeyshare()
	k.Cancel()
}

// Ready returns true if all parties from the peerstore are ready.
// Error is returned if excluded peers exist as we need all peers to participate
// in keygen process.
func (k *Keygen) Ready(readyMap map[peer.ID]bool, excludedPeers []peer.ID) (bool, error) {
	if len(excludedPeers) > 0 {
		return false, errors.New("error")
	}

	return len(readyMap) == len(k.Host.Peerstore().Peers()), nil
}

// ValidCoordinators returns all peers in peerstore
func (k *Keygen) ValidCoordinators() []peer.ID {
	return k.Host.Peerstore().Peers()
}

func (k *Keygen) StartParams(readyMap map[peer.ID]bool) []byte {
	return []byte{}
}

// processEndMessage waits for the final message with generated key share and stores it locally.
func (k *Keygen) processEndMessage(ctx context.Context, endChn chan keygen.LocalPartySaveData) {
	ticker := time.NewTicker(k.Timeout)
	for {
		select {
		case key := <-endChn:
			{
				k.Log.Info().Msgf("Generated key share for address: %s", crypto.PubkeyToAddress(*key.ECDSAPub.ToECDSAPubKey()))

				keyshare := keyshare.NewKeyshare(key, k.threshold, k.Peers)
				err := k.storer.StoreKeyshare(keyshare)
				if err != nil {
					k.ErrChn <- err
				}

				k.ErrChn <- nil
				return
			}
		case <-ticker.C:
			{
				k.ErrChn <- fmt.Errorf("keygen process timed out in: %s", KeygenTimeout)
				return
			}
		case <-ctx.Done():
			{
				return
			}
		}
	}
}
