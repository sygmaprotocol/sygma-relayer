package keygen

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime/debug"
	"sort"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/keyshare"
	"github.com/ChainSafe/sygma-relayer/tss/ecdsa/common"
	"github.com/binance-chain/tss-lib/tss"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sourcegraph/conc/pool"
	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
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
	handler        *protocol.MultiHandler
	done           chan bool
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
		done:      make(chan bool),
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
	// k.storer.LockKeyshare()
	defer k.Stop()

	outChn := make(chan tss.Message)
	msgChn := make(chan *comm.WrappedMessage)
	k.subscriptionID = k.Communication.Subscribe(k.SessionID(), comm.TssKeyGenMsg, msgChn)

	var err error
	k.handler, err = protocol.NewMultiHandler(
		frost.KeygenTaproot(
			party.ID(k.Host.ID().Pretty()),
			PartyIDSFromPeers(append(k.Host.Peerstore().Peers(), k.Host.ID())),
			k.threshold),
		[]byte(k.SessionID()))
	if err != nil {
		return err
	}

	p := pool.New().WithContext(ctx).WithCancelOnError()
	p.Go(func(ctx context.Context) error { return k.ProcessOutboundMessages(ctx, outChn, comm.TssKeyGenMsg) })
	p.Go(func(ctx context.Context) error { return k.ProcessInboundMessages(ctx, msgChn) })
	p.Go(func(ctx context.Context) error { return k.processEndMessage(ctx) })
	return p.Wait()
}

// Stop ends all subscriptions created when starting the tss process and unlocks keyshare.
func (k *Keygen) Stop() {
	k.Communication.UnSubscribe(k.subscriptionID)
	// k.storer.UnlockKeyshare()
	// k.handler.Stop()
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

func (k *Keygen) Retryable() bool {
	return false
}

// processEndMessage waits for the final message with generated key share and stores it locally.
func (k *Keygen) processEndMessage(ctx context.Context) error {

	for {
		select {
		case <-k.done:
			{
				result, err := k.handler.Result()
				if err != nil {
					return err
				}
				taprootConfig := result.(*frost.TaprootConfig)

				k.Log.Info().Msgf("Generated public key %s", hex.EncodeToString(taprootConfig.PublicKey))
				k.Cancel()
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// ProcessInboundMessages processes messages from tss parties and updates local party accordingly.
func (k *Keygen) ProcessInboundMessages(ctx context.Context, msgChan chan *comm.WrappedMessage) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf(string(debug.Stack()))
		}
	}()

	for {
		select {
		case wMsg := <-msgChan:
			{
				k.Log.Debug().Msgf("processed inbound message from %s", wMsg.From)

				msg := &protocol.Message{}
				err := msg.UnmarshalBinary(wMsg.Payload)
				if err != nil {
					return err
				}

				if !k.handler.CanAccept(msg) {
					continue
				}
				k.handler.Accept(msg)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// ProcessOutboundMessages sends messages received from tss out channel to target peers.
// On context cancel stops listening to channel and exits.
func (k *Keygen) ProcessOutboundMessages(ctx context.Context, outChn chan tss.Message, messageType comm.MessageType) error {
	for {
		select {
		case msg, ok := <-k.handler.Listen():
			{
				if !ok {
					k.done <- true
					return nil
				}

				msgBytes, err := msg.MarshalBinary()
				if err != nil {
					return err
				}

				peers, err := k.BroadcastPeers(msg)
				if err != nil {
					return err
				}

				k.Log.Debug().Msgf("sending message to %s", peers)
				err = k.Communication.Broadcast(peers, msgBytes, messageType, k.SessionID())
				if err != nil {
					return err
				}
			}
		case <-ctx.Done():
			{
				return nil
			}
		}
	}
}

func (k *Keygen) BroadcastPeers(msg *protocol.Message) ([]peer.ID, error) {
	if msg.Broadcast {
		return k.Peers, nil
	} else {
		p, err := peer.Decode(string(msg.To))
		if err != nil {
			return nil, err
		}
		return []peer.ID{p}, nil
	}
}

func PartyIDSFromPeers(peers peer.IDSlice) []party.ID {
	sort.Sort(peers)
	peerSet := mapset.NewSet[peer.ID](peers...)
	idSlice := make([]party.ID, len(peerSet.ToSlice()))
	for i, peer := range peerSet.ToSlice() {
		idSlice[i] = party.ID(peer.Pretty())
	}
	return idSlice
}
