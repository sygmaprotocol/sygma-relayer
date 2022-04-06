package keygen

import (
	"context"
	"fmt"
	"time"

	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/chainbridge-core/tss/common"
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
	StoreKeyshare(keyshare store.Keyshare) error
}

type Keygen struct {
	common.BaseTss
	storer    SaveDataStorer
	threshold int
}

func NewKeygen(
	sessionID string,
	threshold int,
	host host.Host,
	comm common.Communication,
	storer SaveDataStorer,
	errChn chan error,
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
			ErrChn:        errChn,
			Timeout:       KeygenTimeout,
		},
		storer:    storer,
		threshold: threshold,
	}
}

// Start initializes the keygen party and starts the keygen tss process.
//
// Should be run only after all the participating parties are ready.
func (k *Keygen) Start(ctx context.Context, params []string) {
	parties := common.PartiesFromPeers(k.Host.Peerstore().Peers())
	k.PopulatePartyStore(parties)

	pCtx := tss.NewPeerContext(parties)
	tssParams := tss.NewParameters(pCtx, k.PartyStore[k.Host.ID().Pretty()], len(parties), k.threshold)

	outChn := make(chan tss.Message)
	msgChn := make(chan *common.WrappedMessage)
	endChn := make(chan keygen.LocalPartySaveData)

	k.Communication.Subscribe(common.KeyGenMsg, k.SessionID(), msgChn)

	go k.ProcessOutboundMessages(ctx, outChn, common.KeyGenMsg)
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

// Stop ends all subscriptions created when starting the tss process.
func (k *Keygen) Stop() {
	k.Communication.CancelSubscribe(common.KeyGenMsg, k.SessionID())
}

// Ready returns true if all parties from the peerstore are ready.
func (k *Keygen) Ready(readyMap map[peer.ID]bool) bool {
	return len(readyMap) == len(k.Host.Peerstore().Peers())
}

// StartParams returns params necessary to start the tss process which
// are sent in the start message.
func (k *Keygen) StartParams() []string {
	return []string{}
}

// processEndMessage waits for the final message with generated key share and stores it locally.
func (k *Keygen) processEndMessage(ctx context.Context, endChn chan keygen.LocalPartySaveData) {
	ticker := time.NewTicker(k.Timeout)
	for {
		select {
		case key := <-endChn:
			{
				k.Log.Info().Msgf("Generated key share for address: %s", crypto.PubkeyToAddress(*key.ECDSAPub.ToECDSAPubKey()))

				keyshare := store.NewKeyshare(key, k.threshold, k.Peers)
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
