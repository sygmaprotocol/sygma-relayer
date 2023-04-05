package substrate

import (
	"bytes"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func ExtrinsicHash(ext types.Extrinsic) (string, error) {
	extHash := bytes.NewBuffer([]byte{})
	encoder := scale.NewEncoder(extHash)
	err := ext.Encode(*encoder)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(extHash.Bytes()), nil
}
