package substrate

import (
	"github.com/centrifuge/go-substrate-rpc-client/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

type MultiLocationV3 struct {
	Parents  types.U8
	Interior JunctionsV3
}

func (m *MultiLocationV3) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&m.Parents); err != nil {
		return err
	}

	return decoder.Decode(&m.Interior)
}

func (m *MultiLocationV3) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(&m.Parents); err != nil {
		return err
	}

	return encoder.Encode(&m.Interior)
}
