package executor

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ChainSafe/sygma-relayer/chains/btc/connection"
	"github.com/ChainSafe/sygma-relayer/chains/btc/mempool"
	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/tss"
	"github.com/ChainSafe/sygma-relayer/tss/frost/signing"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
	"github.com/sourcegraph/conc/pool"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
	"github.com/taurusgroup/multi-party-sig/pkg/taproot"
)

var (
	executionCheckPeriod = time.Minute
	signingTimeout       = 30 * time.Minute

	INPUT_SIZE  = 180
	OUTPUT_SIZE = 34
)

type MempoolAPI interface {
	RecommendedFee() (*mempool.Fee, error)
	Utxos(address string) ([]mempool.Utxo, error)
}

type Executor struct {
	coordinator   *tss.Coordinator
	host          host.Host
	comm          comm.Communication
	fetcher       signing.SaveDataFetcher
	exitLock      *sync.RWMutex
	publicKey     *secp256k1.PublicKey
	conn          *connection.Connection
	senderAddress btcutil.Address
	chainCfg      chaincfg.Params
	mempool       MempoolAPI
}

func NewExecutor(
	host host.Host,
	comm comm.Communication,
	coordinator *tss.Coordinator,
	fetcher signing.SaveDataFetcher,
	conn *connection.Connection,
	mempool MempoolAPI,
	exitLock *sync.RWMutex,
) *Executor {
	taprootAddress, _ := btcutil.DecodeAddress("tb1pdf5c3q35ssem2l25n435fa69qr7dzwkc6gsqehuflr3euh905l2slafjvv", &chaincfg.TestNet3Params)
	// taprootAddress, _ := btcutil.DecodeAddress("tb1pmam5aqdlr3m4dc6uswzwx6dl6yyjamkrrdj7eq422vh2ny9spvjq5d8jfa", &chaincfg.TestNet3Params)
	return &Executor{
		host:          host,
		comm:          comm,
		coordinator:   coordinator,
		exitLock:      exitLock,
		fetcher:       fetcher,
		conn:          conn,
		senderAddress: taprootAddress,
		mempool:       mempool,
		chainCfg:      chaincfg.TestNet3Params,
	}
}

// Execute starts a signing process and executes proposals when signature is generated
func (e *Executor) Execute(props []*proposal.Proposal) error {
	e.exitLock.RLock()
	defer e.exitLock.RUnlock()

	sessionID := props[0].MessageID
	proposals := make([]*proposal.Proposal, 0)
	for _, prop := range props {
		/* TODO
		propStatus, err := e.propStore.GetPropStatus(prop.Source, prop.Destination, prop.Data.())
		if err != nil {
			return err
		}
		if propStatus == store.MissingProp {
			continue
		}
		*/

		proposals = append(proposals, prop)
	}
	if len(proposals) == 0 {
		return nil
	}

	tx, _, err := e.rawTx(proposals)
	if err != nil {
		return err
	}

	script, _ := hex.DecodeString("51206a698882348433b57d549d6344f74500fcd13ad8d2200cdf89f8e39e5cafa7d5")
	prevOutputFetcher := txscript.NewCannedPrevOutputFetcher(script, 13918)
	sigHash := txscript.NewTxSigHashes(tx, prevOutputFetcher)
	txHash, err := txscript.CalcTaprootSignatureHash(sigHash, txscript.SigHashDefault, tx, 0, prevOutputFetcher)
	if err != nil {
		return err
	}

	p := pool.New().WithErrors()
	p.Go(func() error {
		msg := new(big.Int)
		msg.SetBytes(txHash[:])
		signing, err := signing.NewSigning(
			msg,
			sessionID,
			e.host,
			e.comm,
			e.fetcher)
		if err != nil {
			return err
		}

		sigChn := make(chan interface{})
		executionContext, cancelExecution := context.WithCancel(context.Background())
		watchContext, cancelWatch := context.WithCancel(context.Background())
		ep := pool.New().WithErrors()
		ep.Go(func() error {
			err := e.coordinator.Execute(executionContext, signing, sigChn)
			if err != nil {
				cancelWatch()
			}

			return err
		})
		ep.Go(func() error { return e.watchExecution(watchContext, cancelExecution, tx, proposals, sigChn, sessionID) })
		return ep.Wait()
	})
	return p.Wait()
}

func (e *Executor) watchExecution(ctx context.Context, cancelExecution context.CancelFunc, tx *wire.MsgTx, proposals []*proposal.Proposal, sigChn chan interface{}, sessionID string) error {
	ticker := time.NewTicker(executionCheckPeriod)
	timeout := time.NewTicker(signingTimeout)
	defer ticker.Stop()
	defer timeout.Stop()
	defer cancelExecution()

	for {
		select {
		case sigResult := <-sigChn:
			{
				cancelExecution()
				if sigResult == nil {
					continue
				}

				signatureData := sigResult.(taproot.Signature)
				hash, err := e.sendTx(tx, signatureData)
				if err != nil {
					_ = e.comm.Broadcast(e.host.Peerstore().Peers(), []byte{}, comm.TssFailMsg, sessionID)
					return err
				}

				log.Info().Str("SessionID", sessionID).Msgf("Sent proposals execution with hash: %s", hash)
			}
		case <-ticker.C:
			{
				if !e.areProposalsExecuted(proposals, sessionID) {
					continue
				}

				log.Info().Str("SessionID", sessionID).Msgf("Successfully executed proposals")
				return nil
			}
		case <-timeout.C:
			{
				return fmt.Errorf("execution timed out in %s", signingTimeout)
			}
		case <-ctx.Done():
			{
				return nil
			}
		}
	}
}

func (e *Executor) rawTx(proposals []*proposal.Proposal) (*wire.MsgTx, []byte, error) {
	utxos, err := e.mempool.Utxos(e.senderAddress.String())
	if err != nil {
		return nil, nil, err
	}
	if len(utxos) == 0 {
		return nil, nil, fmt.Errorf("no utxos found")
	}
	var utxo mempool.Utxo
	for _, u := range utxos {
		if u.Value > 3000 {
			utxo = u
			break
		}
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	previousTxHash, err := chainhash.NewHashFromStr(utxo.TxID)
	if err != nil {
		return nil, nil, err
	}
	outPoint := wire.NewOutPoint(previousTxHash, utxo.Vout)
	txIn := wire.NewTxIn(outPoint, nil, nil)
	tx.AddTxIn(txIn)

	totalAmount := int64(0)
	for _, prop := range proposals {
		propData := prop.Data.(BtcProposalData)
		addr, err := btcutil.DecodeAddress(propData.Recipient, &e.chainCfg)
		if err != nil {
			return nil, nil, err
		}
		destinationAddrByte, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, nil, err
		}
		txOut := wire.NewTxOut(propData.Amount, destinationAddrByte)
		tx.AddTxOut(txOut)
		totalAmount += propData.Amount
	}

	// return extra funds
	returnScript, err := txscript.PayToAddrScript(e.senderAddress)
	if err != nil {
		return nil, nil, err
	}
	txOut := wire.NewTxOut(int64(utxo.Value)-3000-totalAmount, returnScript)
	tx.AddTxOut(txOut)

	return tx, nil, err
}

func (e *Executor) fee(numOfInputs, numOfOutputs int64) (int64, error) {
	recommendedFee, err := e.mempool.RecommendedFee()
	if err != nil {
		return 0, err
	}
	fmt.Println(recommendedFee.FastestFee)
	return 3000, nil
}

func (e *Executor) sendTx(tx *wire.MsgTx, signature taproot.Signature) (*chainhash.Hash, error) {
	tx.TxIn[0].Witness = wire.TxWitness{signature}
	return e.conn.SendRawTransaction(tx, true)
}

// TODO
func (e *Executor) areProposalsExecuted(proposals []*proposal.Proposal, sessionID string) bool {
	return true
}

func BytesToModNScalar(data []byte) *secp256k1.ModNScalar {
	scalar := new(secp256k1.ModNScalar)
	scalar.SetByteSlice(data)
	return scalar
}
