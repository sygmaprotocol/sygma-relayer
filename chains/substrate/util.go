package substrate

import (
	"bytes"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

func ExtrinsicHash(ext types.Extrinsic) (types.Hash, error) {
	extHash := bytes.NewBuffer([]byte{})
	encoder := scale.NewEncoder(extHash)
	err := ext.Encode(*encoder)
	if err != nil {
		return types.Hash{}, err
	}

	return types.NewHash(extHash.Bytes()), nil
}
