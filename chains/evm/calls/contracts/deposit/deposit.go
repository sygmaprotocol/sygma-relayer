package deposit

import (
	"bytes"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/types"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common/math"
)

func constructMainDepositData(tokenStats *big.Int, destRecipient []byte) []byte {
	var data []byte
	data = append(data, math.PaddedBigBytes(tokenStats, 32)...)                            // Amount (ERC20) or Token Id (ERC721)
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(len(destRecipient))), 32)...) // length of recipient
	data = append(data, destRecipient...)                                                  // Recipient
	return data
}

func ConstructFeeData(baseRate, tokenRate string, destGasPrice *big.Int, expirationTimestamp int64, fromDomainId, toDomainId uint8,
	resourceID types.ResourceID, tokenDecimal int64, baseCurrencyDecimal int64, feeOracleSignature []byte, amount *big.Int) ([]byte, error) {

	ber, err := calls.UserAmountToWei(baseRate, big.NewInt(baseCurrencyDecimal))
	if err != nil {
		return nil, err
	}
	finalBaseEffectiveRate := calls.PaddingZero(ber.Bytes(), 32)
	ter, err := calls.UserAmountToWei(tokenRate, big.NewInt(tokenDecimal))
	if err != nil {
		return nil, err
	}
	finalTokenEffectiveRate := calls.PaddingZero(ter.Bytes(), 32)

	finalGasPrice := calls.PaddingZero(destGasPrice.Bytes(), 32)
	finalTimestamp := calls.PaddingZero([]byte(strconv.FormatInt(expirationTimestamp, 16)), 32)
	finalFromDomainId := calls.PaddingZero([]byte{fromDomainId}, 32)
	finalToDomainId := calls.PaddingZero([]byte{toDomainId}, 32)

	feeDataMessageByte := bytes.Buffer{}
	feeDataMessageByte.Write(finalBaseEffectiveRate)
	feeDataMessageByte.Write(finalTokenEffectiveRate)
	feeDataMessageByte.Write(finalGasPrice)
	feeDataMessageByte.Write(finalTimestamp)
	feeDataMessageByte.Write(finalFromDomainId)
	feeDataMessageByte.Write(finalToDomainId)
	feeDataMessageByte.Write(calls.Bytes32ToSlice(resourceID))
	finalFeeDataMessage := feeDataMessageByte.Bytes()

	finalAmount := calls.PaddingZero(amount.Bytes(), 32)

	feeData := bytes.Buffer{}
	feeData.Write(finalFeeDataMessage)
	feeData.Write(feeOracleSignature)
	feeData.Write(finalAmount)

	return feeData.Bytes(), nil
}

func ConstructErc20DepositData(destRecipient []byte, amount *big.Int) []byte {
	data := constructMainDepositData(amount, destRecipient)
	return data
}

func ConstructErc20DepositDataWithPriority(destRecipient []byte, amount *big.Int, priority uint8) []byte {
	data := constructMainDepositData(amount, destRecipient)
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(len([]uint8{priority}))), 1)...) // Length of priority
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(priority)), 1)...)               // Priority
	return data
}

func ConstructErc721DepositData(destRecipient []byte, tokenId *big.Int, metadata []byte) []byte {
	data := constructMainDepositData(tokenId, destRecipient)
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(len(metadata))), 32)...) // Length of metadata
	data = append(data, metadata...)                                                  // Metadata
	return data
}

func ConstructErc721DepositDataWithPriority(destRecipient []byte, tokenId *big.Int, metadata []byte, priority uint8) []byte {
	data := constructMainDepositData(tokenId, destRecipient)
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(len(metadata))), 32)...)         // Length of metadata
	data = append(data, metadata...)                                                          // Metadata
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(len([]uint8{priority}))), 1)...) // Length of priority
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(priority)), 1)...)               // Priority
	return data
}

func ConstructGenericDepositData(metadata []byte) []byte {
	var data []byte
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(len(metadata))), 32)...) // Length of metadata
	data = append(data, metadata...)                                                  // Metadata
	return data
}
