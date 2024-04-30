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
	publicKey *secp256k1.PublicKey,
	exitLock *sync.RWMutex,
) *Executor {
	addr, _ := btcutil.DecodeAddress("mrgYN96jj1bPiFdyAtP9pEezzjfQc5YdaQ", &chaincfg.TestNet3Params)
	return &Executor{
		host:          host,
		comm:          comm,
		coordinator:   coordinator,
		exitLock:      exitLock,
		fetcher:       fetcher,
		conn:          conn,
		publicKey:     publicKey,
		senderAddress: addr,
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
	prevOuts := make(map[wire.OutPoint]*wire.TxOut)
	prevFetcher := txscript.NewMultiPrevOutFetcher(prevOuts)
	sigHash := txscript.NewTxSigHashes(tx, prevFetcher)

	script, _ := hex.DecodeString("76a92063f4b4ab0bb1d6a364bde810aead8dd571cea18586e2a84d6a640e6ff80cb42188ac")
	leaf := txscript.NewBaseTapLeaf(script)
	txHash, err := txscript.CalcTapscriptSignaturehash(sigHash, txscript.SigHashDefault, tx, 0, prevFetcher, leaf)
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

				signatureData := sigResult.(*taproot.Signature)
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
	utxo := utxos[0]

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

	/* TODO
	// return extra funds
	returnScript, err := txscript.PayToAddrScript(e.senderAddress)
	if err != nil {
		return nil, nil, err
	}
	fee, err := e.fee(1, int64(len(proposals)))
	if err != nil {
		return nil, nil, err
	}
	txOut := wire.NewTxOut(int64(utxo.Value)-fee-totalAmount, returnScript)
	tx.AddTxOut(txOut)
	*/

	return tx, nil, err
}

func (e *Executor) fee(numOfInputs, numOfOutputs int64) (int64, error) {
	recommendedFee, err := e.mempool.RecommendedFee()
	if err != nil {
		return 0, err
	}
	return (numOfInputs*int64(INPUT_SIZE) + numOfOutputs*int64(OUTPUT_SIZE) + 10) * recommendedFee.FastestFee, nil
}

func (e *Executor) sendTx(tx *wire.MsgTx, signature *taproot.Signature) (*chainhash.Hash, error) {
	// control block
	/*
		script, _ := hex.DecodeString("76a92063f4b4ab0bb1d6a364bde810aead8dd571cea18586e2a84d6a640e6ff80cb42188ac")
		leaf := txscript.NewBaseTapLeaf(script)
		indexedTree := txscript.AssembleTaprootScriptTree(leaf, leaf)
		settleMerkleProof := indexedTree.LeafMerkleProofs[0]
		pubKeyBytes, _ := hex.DecodeString("63f4b4ab0bb1d6a364bde810aead8dd571cea18586e2a84d6a640e6ff80cb421")
		internalKey, err := secp256k1.ParsePubKey(pubKeyBytes)
		if err != nil {
			return nil, err
		}
		cb := settleMerkleProof.ToControlBlock(internalKey)
		cbBytes, _ := cb.ToBytes()
	*/

	tx.TxIn[0].Witness = wire.TxWitness{*signature}
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
