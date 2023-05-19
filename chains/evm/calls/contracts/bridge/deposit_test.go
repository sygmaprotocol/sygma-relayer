package bridge

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
)

func TestConstructPermissionlessGenericDepositData(t *testing.T) {
	type args struct {
		metadata               []byte
		executionFunctionSig   []byte
		executeContractAddress []byte
		metadataDepositor      []byte
		maxFee                 *big.Int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "test",
			args: args{},
			want: []byte{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := hex.DecodeString("0102030405060708")
			fmt.Println(err)
			functionSig, _ := hex.DecodeString("12345678")
			contractAddress, err := hex.DecodeString("abcdef1234567890")
			fmt.Println(err)
			maxFee, _ := hex.DecodeString("1000")
			maxFeeBig := new(big.Int).SetBytes(maxFee)
			depositor, _ := hex.DecodeString("1234abcd5678ef90")
			got := ConstructPermissionlessGenericDepositData(metadata, functionSig, contractAddress, depositor, maxFeeBig)
			fmt.Println(hex.EncodeToString(got))
			t.Fail()
		})
	}
}
