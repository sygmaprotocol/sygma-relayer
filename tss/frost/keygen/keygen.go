package keygen

import (
	"context"
	"errors"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/keyshare"
	"github.com/ChainSafe/sygma-relayer/tss/ecdsa/common"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-kit/kit/log"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
)

type SaveDataStorer interface {
	StoreKeyshare(keyshare keyshare.Keyshare) error
	LockKeyshare()
	UnlockKeyshare()
	GetKeyshare() (keyshare.Keyshare, error)
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
			Cancel:        func() {},
		},
		storer:    storer,
		threshold: threshold,
	}
}

// Run initializes the keygen party and runs the keygen tss process.
//
// Should be run only after all the participating parties are ready.
func (k *Keygen) Run(
	ctx context.Context,
	coordinator bool,
	resultChn chan interface{},
	params []byte,
) error {
	ctx, k.Cancel = context.WithCancel(ctx)
	k.storer.LockKeyshare()

	outChn := make(chan tss.Message)
	msgChn := make(chan *comm.WrappedMessage)
	endChn := make(chan keygen.LocalPartySaveData)
	k.subscriptionID = k.Communication.Subscribe(k.SessionID(), comm.TssKeyGenMsg, msgChn)

	startFunc := frost.KeygenTaproot(party.ID(k.Host.ID()), PartyIDSFromPeers(k.Host.Peerstore().Peers()), k.threshold)
	startFunc([]byte(k.SessionID()))
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
func (k *Keygen) processEndMessage(ctx context.Context, endChn chan keygen.LocalPartySaveData) error {
	defer k.Cancel()
	for {
		select {
		case key := <-endChn:
			{
				k.Log.Info().Msgf("Generated key share for address: %s", crypto.PubkeyToAddress(*key.ECDSAPub.ToBtcecPubKey().ToECDSA()))

				keyshare := keyshare.NewKeyshare(key, k.threshold, k.Peers)
				err := k.storer.StoreKeyshare(keyshare)
				if err != nil {
					return err
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

func (k *Keygen) Retryable() bool {
	return false
}

func PartyIDSFromPeers(peers peer.IDSlice) []party.ID {
	idSlice := make([]party.ID, len(peers))
	for i, peer := range peers {
		idSlice[i] = party.ID(peer.Pretty())
	}
	return idSlice
}
