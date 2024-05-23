package listener

import (
	"encoding/hex"
	"math/big"

	"github.com/ChainSafe/sygma-relayer/chains/btc"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
	PubKeyHash       = "pubkeyhash"
	ScriptHash       = "scripthash"
	WitnessV0KeyHash = "witness_v0_keyhash"
	OP_RETURN        = "nulldata"
)

func DecodeDepositEvent(evt btcjson.TxRawResult, conn Connection, resource btc.ResourceConfig) (Deposit, bool, error) {
	amount := big.NewInt(0)
	isBridgeDeposit := false
	sender := ""
	data := ""
	resourceID := [32]byte{}
	for _, vout := range evt.Vout {
		// read the OP_RETURN data
		if vout.ScriptPubKey.Type == OP_RETURN {
			opReturnData, err := hex.DecodeString(vout.ScriptPubKey.Hex)
			if err != nil {
				return Deposit{}, true, err
			}
			// Extract OP_RETURN data (excluding OP_RETURN prefix)
			data = string(opReturnData[2:])
		}

		if resource.Address == vout.ScriptPubKey.Address {
			isBridgeDeposit = true
			resourceID = resource.ResourceID
			if vout.ScriptPubKey.Type == PubKeyHash || vout.ScriptPubKey.Type == ScriptHash || vout.ScriptPubKey.Type == WitnessV0KeyHash {
				amount.Add(amount, big.NewInt(int64(vout.Value*1e8)))
			}
		}
	}

	if !isBridgeDeposit {
		return Deposit{}, false, nil
	}

	for _, vin := range evt.Vin {

		// Retrieve the previous transaction output
		hsh, err := chainhash.NewHashFromStr(vin.Txid)
		if err != nil {
			return Deposit{}, true, err
		}
		prevTx, err := conn.GetRawTransactionVerbose(hsh)
		if err != nil {
			return Deposit{}, true, err
		}

		// Retrieve the sender's address from the scriptPubKey of the previous output
		sender = prevTx.Vout[vin.Vout].ScriptPubKey.Address
	}

	return Deposit{
		ResourceID:    resourceID,
		SenderAddress: sender,
		Amount:        amount,
		Data:          data,
	}, true, nil
}
