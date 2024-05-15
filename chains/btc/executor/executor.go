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
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
	"github.com/sourcegraph/conc/pool"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
	"github.com/taurusgroup/multi-party-sig/pkg/taproot"
	"go.uber.org/zap/buffer"
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
	conn          *connection.Connection
	senderAddress btcutil.Address
	tweak         string
	script        []byte
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
	address btcutil.Address,
	tweak string,
	script []byte,
	chainCfg chaincfg.Params,
	exitLock *sync.RWMutex,
) *Executor {
	return &Executor{
		host:          host,
		comm:          comm,
		coordinator:   coordinator,
		exitLock:      exitLock,
		fetcher:       fetcher,
		conn:          conn,
		senderAddress: address,
		tweak:         tweak,
		script:        script,
		mempool:       mempool,
		chainCfg:      chainCfg,
	}
}

// Execute starts a signing process and executes proposals when signature is generated
func (e *Executor) Execute(props []*proposal.Proposal) error {
	e.exitLock.RLock()
	defer e.exitLock.RUnlock()

	if len(props) == 0 {
		return nil
	}
	sessionID := props[0].MessageID

	tx, utxos, err := e.rawTx(props)
	if err != nil {
		return err
	}

	sigChn := make(chan interface{})
	p := pool.New().WithErrors()
	executionContext, cancelExecution := context.WithCancel(context.Background())
	watchContext, cancelWatch := context.WithCancel(context.Background())
	defer cancelWatch()
	p.Go(func() error { return e.watchExecution(watchContext, cancelExecution, tx, props, sigChn, sessionID) })
	prevOuts := make(map[wire.OutPoint]*wire.TxOut)
	for _, utxo := range utxos {
		txOut := wire.NewTxOut(int64(utxo.Value), e.script)
		hash, err := chainhash.NewHashFromStr(utxo.TxID)
		if err != nil {
			return err
		}
		prevOuts[*wire.NewOutPoint(hash, utxo.Vout)] = txOut
	}
	prevOutputFetcher := txscript.NewMultiPrevOutFetcher(prevOuts)
	sigHashes := txscript.NewTxSigHashes(tx, prevOutputFetcher)

	// we need to sign each input individually
	for i := range tx.TxIn {
		txHash, err := txscript.CalcTaprootSignatureHash(sigHashes, txscript.SigHashDefault, tx, i, prevOutputFetcher)
		if err != nil {
			return err
		}
		p.Go(func() error {
			msg := new(big.Int)
			msg.SetBytes(txHash[:])
			signing, err := signing.NewSigning(
				i,
				msg,
				e.tweak,
				fmt.Sprintf("%s-%d", sessionID, i),
				e.host,
				e.comm,
				e.fetcher)
			if err != nil {
				return err
			}
			return e.coordinator.Execute(executionContext, signing, sigChn)
		})
	}
	return p.Wait()
}

func (e *Executor) watchExecution(ctx context.Context, cancelExecution context.CancelFunc, tx *wire.MsgTx, proposals []*proposal.Proposal, sigChn chan interface{}, sessionID string) error {
	ticker := time.NewTicker(executionCheckPeriod)
	timeout := time.NewTicker(signingTimeout)
	defer ticker.Stop()
	defer timeout.Stop()
	defer cancelExecution()
	signatures := make([]taproot.Signature, len(tx.TxIn))

	for {
		select {
		case sigResult := <-sigChn:
			{
				if sigResult == nil {
					continue
				}
				signatureData := sigResult.(signing.Signature)
				signatures[signatureData.Id] = signatureData.Signature
				if !e.signaturesFilled(signatures) {
					continue
				}
				cancelExecution()

				hash, err := e.sendTx(tx, signatures)
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

func (e *Executor) rawTx(proposals []*proposal.Proposal) (*wire.MsgTx, []mempool.Utxo, error) {
	tx := wire.NewMsgTx(wire.TxVersion)
	outputAmount, err := e.outputs(tx, proposals)
	if err != nil {
		return nil, nil, err
	}
	inputAmount, utxos, err := e.inputs(tx, outputAmount)
	if err != nil {
		return nil, nil, err
	}
	if inputAmount < outputAmount {
		return nil, nil, fmt.Errorf("utxo input amount %d less than output amount %d", inputAmount, outputAmount)
	}
	fee, err := e.fee(int64(len(utxos)), int64(len(proposals))+1)
	if err != nil {
		return nil, nil, err
	}

	returnAmount := int64(inputAmount) - fee - outputAmount
	if returnAmount > 0 {
		// return extra funds
		returnScript, err := txscript.PayToAddrScript(e.senderAddress)
		if err != nil {
			return nil, nil, err
		}
		txOut := wire.NewTxOut(returnAmount, returnScript)
		tx.AddTxOut(txOut)
	}
	return tx, utxos, err
}

func (e *Executor) outputs(tx *wire.MsgTx, proposals []*proposal.Proposal) (int64, error) {
	outputAmount := int64(0)
	for _, prop := range proposals {
		propData := prop.Data.(BtcTransferProposalData)
		addr, err := btcutil.DecodeAddress(propData.Recipient, &e.chainCfg)
		if err != nil {
			return 0, err
		}
		destinationAddrByte, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return 0, err
		}
		txOut := wire.NewTxOut(propData.Amount, destinationAddrByte)
		tx.AddTxOut(txOut)
		outputAmount += propData.Amount
	}
	return outputAmount, nil
}

func (e *Executor) inputs(tx *wire.MsgTx, outputAmount int64) (int64, []mempool.Utxo, error) {
	usedUtxos := make([]mempool.Utxo, 0)
	inputAmount := int64(0)
	utxos, err := e.mempool.Utxos(e.senderAddress.String())
	if err != nil {
		return 0, nil, err
	}
	for _, utxo := range utxos {
		previousTxHash, err := chainhash.NewHashFromStr(utxo.TxID)
		if err != nil {
			return 0, nil, err
		}
		outPoint := wire.NewOutPoint(previousTxHash, utxo.Vout)
		txIn := wire.NewTxIn(outPoint, nil, nil)
		tx.AddTxIn(txIn)

		usedUtxos = append(usedUtxos, utxo)
		inputAmount += int64(utxo.Value)
		if inputAmount > outputAmount {
			break
		}
	}
	return inputAmount, usedUtxos, nil
}

func (e *Executor) fee(numOfInputs, numOfOutputs int64) (int64, error) {
	recommendedFee, err := e.mempool.RecommendedFee()
	if err != nil {
		return 0, err
	}
	return (numOfInputs*int64(INPUT_SIZE) + numOfOutputs*int64(OUTPUT_SIZE)) * recommendedFee.EconomyFee, nil
}

func (e *Executor) sendTx(tx *wire.MsgTx, signatures []taproot.Signature) (*chainhash.Hash, error) {
	for i, sig := range signatures {
		tx.TxIn[i].Witness = wire.TxWitness{sig}
	}

	var buf buffer.Buffer
	err := tx.Serialize(&buf)
	if err != nil {
		return nil, err
	}
	bytes := buf.Bytes()
	log.Debug().Msgf("Assembled raw transaction %s", hex.EncodeToString(bytes))
	return e.conn.SendRawTransaction(tx, true)
}

func (e *Executor) signaturesFilled(signatures []taproot.Signature) bool {
	for _, signature := range signatures {
		if len([]byte(signature)) == 0 {
			return false
		}
	}

	return true
}

func (e *Executor) areProposalsExecuted(proposals []*proposal.Proposal, sessionID string) bool {
	return false
}
