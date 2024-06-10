package listener

import (
	"encoding/hex"
	"math/big"

	"github.com/ChainSafe/sygma-relayer/chains/btc/config"
	"github.com/btcsuite/btcd/btcjson"
)

const (
	WitnessV1Taproot = "witness_v1_taproot"
	OP_RETURN        = "nulldata"
)

func DecodeDepositEvent(evt btcjson.TxRawResult, resource config.Resource) (Deposit, bool, error) {
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

		if resource.Address.String() == vout.ScriptPubKey.Address {
			isBridgeDeposit = true
			resourceID = resource.ResourceID
			if vout.ScriptPubKey.Type == WitnessV1Taproot {
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

func SliceTo32Bytes(in []byte) [32]byte {
	var res [32]byte
	copy(res[:], in)
	return res
}
