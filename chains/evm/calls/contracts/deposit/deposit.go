package deposit

import (
	"math/big"

	"github.com/ChainSafe/sygma-relayer/chains/evm/listener"
)

func ConstructErc1155DepositData(destRecipient []byte, tokenIds *big.Int, amounts *big.Int, metadata []byte) ([]byte, error) {
	erc1155Type, err := listener.GetErc1155Type()
	if err != nil {
		return nil, err
	}

	payload := []interface{}{
		[]*big.Int{
			tokenIds,
		},
		[]*big.Int{
			amounts,
		},
		destRecipient,
		[]byte{},
	}
	data, err := erc1155Type.Pack(payload...)

	if err != nil {
		return nil, err
	}
	return data, nil
}
