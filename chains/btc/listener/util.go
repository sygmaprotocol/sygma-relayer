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

func DecodeDepositEvent(evt btcjson.TxRawResult, conn rpcclient.Client, bridges map[string]uint8) (Deposit, bool, error) {
	amount := big.NewInt(0)
	isBridgeDeposit := false
	sender := ""
	reciever := ""
	destinationDomainID := uint8(0)

	for _, vout := range evt.Vout {

		if dstDomainID, ok := bridges[vout.ScriptPubKey.Address]; ok {
			isBridgeDeposit = true
			destinationDomainID = dstDomainID
			// need to check this part if i calculate the amount correctly
			if vout.ScriptPubKey.Type == "pubkeyhash" || vout.ScriptPubKey.Type == "scripthash" || vout.ScriptPubKey.Type == "witness_v0_keyhash" {
				amount.Add(amount, big.NewInt(int64(vout.Value)))
			} else if vout.ScriptPubKey.Type == "nulldata" {
				opReturnData, err := hex.DecodeString(vout.ScriptPubKey.Hex)
				if err != nil {
					return Deposit{}, true, err
				}

				// Extract OP_RETURN data (excluding OP_RETURN prefix)
				reciever = string(opReturnData[2:])
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
			DestinationDomainID: destinationDomainID,
			ResourceID:          [32]byte(RESOURCE_ID),
			SenderAddress:       sender,
			Amount:              amount,
			Reciever:            common.HexToAddress(reciever),
		}, true, nil
	}
	return Deposit{}, false, nil
}
