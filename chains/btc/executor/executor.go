package executor

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ChainSafe/sygma-relayer/chains/btc/config"
	"github.com/ChainSafe/sygma-relayer/chains/btc/connection"
	"github.com/ChainSafe/sygma-relayer/chains/btc/mempool"
	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/store"
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
	signingTimeout = 30 * time.Minute

	INPUT_SIZE  = 180
	OUTPUT_SIZE = 34
)

type MempoolAPI interface {
	RecommendedFee() (*mempool.Fee, error)
	Utxos(address string) ([]mempool.Utxo, error)
}

type PropStorer interface {
	StorePropStatus(source, destination uint8, depositNonce uint64, status store.PropStatus) error
	PropStatus(source, destination uint8, depositNonce uint64) (store.PropStatus, error)
}

type Executor struct {
	coordinator *tss.Coordinator
	host        host.Host
	comm        comm.Communication

	conn      *connection.Connection
	resources map[[32]byte]config.Resource
	chainCfg  chaincfg.Params
	mempool   MempoolAPI
	fetcher   signing.SaveDataFetcher

	propStorer PropStorer
	propMutex  sync.Mutex

	exitLock *sync.RWMutex
}

func NewExecutor(
	propStorer PropStorer,
	host host.Host,
	comm comm.Communication,
	coordinator *tss.Coordinator,
	fetcher signing.SaveDataFetcher,
	conn *connection.Connection,
	mempool MempoolAPI,
	resources map[[32]byte]config.Resource,
	chainCfg chaincfg.Params,
	exitLock *sync.RWMutex,
) *Executor {
	return &Executor{
		propStorer:  propStorer,
		host:        host,
		comm:        comm,
		coordinator: coordinator,
		exitLock:    exitLock,
		fetcher:     fetcher,
		conn:        conn,
		resources:   resources,
		mempool:     mempool,
		chainCfg:    chainCfg,
	}
}

// Execute starts a signing process and executes proposals when signature is generated
func (e *Executor) Execute(proposals []*proposal.Proposal) error {
	e.exitLock.RLock()
	defer e.exitLock.RUnlock()

	sessionID := proposals[0].MessageID
	props, err := e.proposalsForExecution(proposals)
	if err != nil {
		return err
	}
	if len(props) == 0 {
		return nil
	}

	propsPerResource := make(map[[32]byte][]*BtcTransferProposal)
	for _, prop := range props {
		propsPerResource[prop.Data.ResourceId] = append(propsPerResource[prop.Data.ResourceId], prop)
	}

	p := pool.New().WithErrors()
	for resourceID, props := range propsPerResource {
		resourceID := resourceID
		props := props

		p.Go(func() error {
			resource, ok := e.resources[resourceID]
			if !ok {
				return fmt.Errorf("no resource for ID %s", hex.EncodeToString(resourceID[:]))
			}

			sessionID := fmt.Sprintf("%s-%s", sessionID, hex.EncodeToString(resourceID[:]))
			return e.executeResourceProps(props, resource, sessionID)
		})
	}
	return p.Wait()
}

func (e *Executor) executeResourceProps(props []*BtcTransferProposal, resource config.Resource, sessionID string) error {
	log.Info().Str("SessionID", sessionID).Msgf("Executing proposals for resource %s", hex.EncodeToString(resource.ResourceID[:]))

	tx, utxos, err := e.rawTx(props, resource)
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
		txOut := wire.NewTxOut(int64(utxo.Value), resource.Script)
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
				resource.Tweak,
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

func (e *Executor) watchExecution(ctx context.Context, cancelExecution context.CancelFunc, tx *wire.MsgTx, proposals []*BtcTransferProposal, sigChn chan interface{}, sessionID string) error {
	timeout := time.NewTicker(signingTimeout)
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
					e.storeProposalsStatus(proposals, store.FailedProp)
					return err
				}

				log.Info().Str("SessionID", sessionID).Msgf("Sent proposals execution with hash: %s", hash)
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

func (e *Executor) rawTx(proposals []*BtcTransferProposal, resource config.Resource) (*wire.MsgTx, []mempool.Utxo, error) {
	tx := wire.NewMsgTx(wire.TxVersion)
	outputAmount, err := e.outputs(tx, proposals)
	if err != nil {
		return nil, nil, err
	}
	feeEstimate, err := e.fee(int64(len(proposals)), int64(len(proposals)))
	if err != nil {
		return nil, nil, err
	}
	inputAmount, utxos, err := e.inputs(tx, resource.Address, outputAmount+feeEstimate)
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
		returnScript, err := txscript.PayToAddrScript(resource.Address)
		if err != nil {
			return nil, nil, err
		}
		txOut := wire.NewTxOut(returnAmount, returnScript)
		tx.AddTxOut(txOut)
	}
	return tx, utxos, err
}

func (e *Executor) outputs(tx *wire.MsgTx, proposals []*BtcTransferProposal) (int64, error) {
	outputAmount := int64(0)
	for _, prop := range proposals {
		addr, err := btcutil.DecodeAddress(prop.Data.Recipient, &e.chainCfg)
		if err != nil {
			return 0, err
		}
		destinationAddrByte, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return 0, err
		}
		txOut := wire.NewTxOut(prop.Data.Amount, destinationAddrByte)
		tx.AddTxOut(txOut)
		outputAmount += prop.Data.Amount
	}
	return outputAmount, nil
}

func (e *Executor) inputs(tx *wire.MsgTx, address btcutil.Address, outputAmount int64) (int64, []mempool.Utxo, error) {
	usedUtxos := make([]mempool.Utxo, 0)
	inputAmount := int64(0)
	utxos, err := e.mempool.Utxos(address.String())
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

func (e *Executor) proposalsForExecution(proposals []*proposal.Proposal) ([]*BtcTransferProposal, error) {
	e.propMutex.Lock()
	props := make([]*BtcTransferProposal, 0)
	for _, prop := range proposals {
		executed, err := e.isExecuted(prop)
		if err != nil {
			return props, err
		}

		if executed {
			log.Info().Msgf("Proposal %s already executed", fmt.Sprintf("%d-%d-%d", prop.Source, prop.Destination, prop.Data.(BtcTransferProposalData).DepositNonce))
			continue
		}

		err = e.propStorer.StorePropStatus(prop.Source, prop.Destination, prop.Data.(BtcTransferProposalData).DepositNonce, store.PendingProp)
		if err != nil {
			return props, err
		}
		props = append(props, &BtcTransferProposal{
			Source:      prop.Source,
			Destination: prop.Destination,
			Data:        prop.Data.(BtcTransferProposalData),
		})
	}
	e.propMutex.Unlock()
	return props, nil
}

func (e *Executor) isExecuted(prop *proposal.Proposal) (bool, error) {
	status, err := e.propStorer.PropStatus(prop.Source, prop.Destination, prop.Data.(BtcTransferProposalData).DepositNonce)
	if err != nil {
		return true, err
	}

	if status == store.MissingProp || status == store.FailedProp {
		return false, nil
	}
	return true, err
}

func (e *Executor) storeProposalsStatus(props []*BtcTransferProposal, status store.PropStatus) {
	e.propMutex.Lock()
	for _, prop := range props {
		err := e.propStorer.StorePropStatus(
			prop.Source,
			prop.Destination,
			prop.Data.DepositNonce,
			status)
		if err != nil {
			log.Err(err).Msgf("Failed storing proposal %+v status %s", prop, status)
		}
	}
	e.propMutex.Unlock()
}
