package bridge

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
)

func ConstructPermissionlessGenericDepositData(metadata []byte, executionFunctionSig []byte, executeContractAddress []byte, metadataDepositor []byte, maxFee *big.Int) []byte {
	var data []byte
	data = append(data, math.PaddedBigBytes(maxFee, 32)...)
	data = append(data, common.LeftPadBytes(big.NewInt(int64(len(executionFunctionSig))).Bytes(), 2)...)
	data = append(data, executionFunctionSig...)
	data = append(data, []byte{byte(len(executeContractAddress))}...)
	data = append(data, executeContractAddress...)
	data = append(data, []byte{byte(len(metadataDepositor))}...)
	data = append(data, metadataDepositor...)
	data = append(data, metadata...)
	return data
}
