package bridge

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
)

func ConstructPermissionlessGenericDepositData(metadata []byte, executionFunctionSig []byte, executeContractAddress []byte, maxFee *big.Int) []byte {
	var data []byte
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(len(metadata))), 32)...) // Length of metadata
	data = append(data, common.LeftPadBytes(executionFunctionSig, 32)...)
	data = append(data, common.LeftPadBytes(executeContractAddress, 32)...)
	data = append(data, math.PaddedBigBytes(maxFee, 32)...)
	data = append(data, metadata...)
	return data
}
