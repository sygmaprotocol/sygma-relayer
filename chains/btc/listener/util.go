package listener

import (
	"encoding/hex"
	"math/big"

	"github.com/ChainSafe/sygma-relayer/chains/btc"
	"github.com/btcsuite/btcd/btcjson"
)

const (
	PubKeyHash       = "pubkeyhash"
	ScriptHash       = "scripthash"
	WitnessV0KeyHash = "witness_v0_keyhash"
	OP_RETURN        = "nulldata"
)

func DecodeDepositEvent(evt btcjson.TxRawResult, conn Connection, resource btc.Resource) (Deposit, bool, error) {
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

	return Deposit{
		ResourceID:    resourceID,
		SenderAddress: sender,
		Amount:        amount,
		Data:          data,
	}, true, nil
}
