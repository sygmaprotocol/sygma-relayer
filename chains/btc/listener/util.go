package listener

import (
	"encoding/hex"
	"math/big"

	"github.com/ChainSafe/sygma-relayer/e2e/evm"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/ethereum/go-ethereum/common"
)

var RESOURCE_ID = evm.SliceTo32Bytes(common.LeftPadBytes([]byte{0}, 31))

const (
	PubKeyHash       = "pubkeyhash"
	ScriptHash       = "scripthash"
	WitnessV0KeyHash = "witness_v0_keyhash"
	OP_RETURN        = "nulldata"
)

func DecodeDepositEvent(evt btcjson.TxRawResult, conn *rpcclient.Client, bridge string) (Deposit, bool, error) {
	amount := big.NewInt(0)
	isBridgeDeposit := false
	sender := ""
	data := ""

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

		if bridge == vout.ScriptPubKey.Address {
			isBridgeDeposit = true

			if vout.ScriptPubKey.Type == PubKeyHash || vout.ScriptPubKey.Type == ScriptHash || vout.ScriptPubKey.Type == WitnessV0KeyHash {
				amount.Add(amount, big.NewInt(int64(vout.Value*1e8)))
			}
		}
	}

	if isBridgeDeposit {
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
			ResourceID:    [32]byte(RESOURCE_ID),
			SenderAddress: sender,
			Amount:        amount,
			Data:          data,
		}, true, nil
	}
	return Deposit{}, false, nil
}
