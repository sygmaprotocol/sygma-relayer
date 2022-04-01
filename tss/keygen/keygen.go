package keygen

import (
	"context"

	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/chainbridge-core/tss/common"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p-core/host"
)

type SaveDataStorer interface {
	StoreKeyshare(keyshare store.Keyshare) error
}

type Keygen struct {
	common.BaseTss
	storer    SaveDataStorer
	threshold int
}

func NewKeygen(sessionID string, threshold int, host host.Host, comm common.Communication, storer SaveDataStorer) *Keygen {
	partyStore := make(map[string]*tss.PartyID)
	return &Keygen{
		BaseTss: common.BaseTss{
			PartyStore:    partyStore,
			Host:          host,
			Communication: comm,
			Peers:         host.Peerstore().Peers(),
			SessionID:     sessionID,
		},
		storer:    storer,
		threshold: threshold,
	}
}

// Start initializes the keygen party and starts the keygen tss process.
//
// Should be run only after all the participating parties are ready.
func (k *Keygen) Start(ctx context.Context, params ...string) error {
	parties := common.PartiesFromPeers(k.Host.Peerstore().Peers())
	k.PopulatePartyStore(parties)

	pCtx := tss.NewPeerContext(parties)
	tssParams := tss.NewParameters(pCtx, k.PartyStore[k.Host.ID().Pretty()], len(parties), k.threshold)

	outChn := make(chan tss.Message)
	msgChn := make(chan *common.WrappedMessage)
	endChn := make(chan keygen.LocalPartySaveData)

	k.Communication.Subscribe(common.KeyGenMsg, k.SessionID, msgChn)

	go k.ProcessOutboundMessages(ctx, outChn, common.KeyGenMsg)
	go k.ProcessInboundMessages(ctx, msgChn)
	go k.processEndMessage(ctx, endChn)

	k.Party = keygen.NewLocalParty(tssParams, outChn, endChn)
	go func() {
		err := k.Party.Start()
		if err != nil {
			k.ErrChn <- err
		}
	}()

	return nil
}

// Stop ends all subscriptions created when starting the tss process
func (k *Keygen) Stop() {
	k.Communication.CancelSubscribe(common.KeyGenMsg, k.SessionID)
}

// processEndMessage waits for the final message with generated key share and stores it locally.
func (k *Keygen) processEndMessage(ctx context.Context, endChn chan keygen.LocalPartySaveData) {
	for {
		select {
		case key := <-endChn:
			{
				keyshare := store.NewKeyshare(key, k.threshold, k.Peers)
				err := k.storer.StoreKeyshare(keyshare)
				if err != nil {
					k.ErrChn <- err
				}

				k.ErrChn <- nil
			}
		case <-ctx.Done():
			{
				return
			}
		}
	}
}
