package substrate

import (
	"github.com/centrifuge/go-substrate-rpc-client/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

type NetworkIDV3 struct {
	IsByGenesis bool
	ByGenesis   types.Hash

	IsByFork          bool
	ByForkBlockNumber types.U64
	ByForkBlockHash   types.Hash

	IsPolkadot bool
	IsKusama   bool
	IsWestend  bool
	IsRococo   bool
	IsWococo   bool

	IsEthereum bool
	Ethereum   types.UCompact

	IsBitcoinCore bool
	IsBitcoinCash bool
}

func (n *NetworkIDV3) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		n.IsByGenesis = true
		return decoder.Decode(&n.ByGenesis)
	case 1:
		n.IsByFork = true

		if err := decoder.Decode(&n.ByForkBlockNumber); err != nil {
			return err
		}

		return decoder.Decode(&n.ByForkBlockHash)
	case 2:
		n.IsPolkadot = true
	case 3:
		n.IsKusama = true
	case 4:
		n.IsWestend = true
	case 5:
		n.IsRococo = true
	case 6:
		n.IsWococo = true
	case 7:
		n.IsEthereum = true
		return decoder.Decode(&n.Ethereum)
	case 8:
		n.IsBitcoinCore = true
	case 9:
		n.IsBitcoinCash = true
	}

	return nil
}

func (n NetworkIDV3) Encode(encoder scale.Encoder) error {
	switch {
	case n.IsByGenesis:
		if err := encoder.PushByte(0); err != nil {
			return err
		}
		return encoder.Encode(n.ByGenesis)
	case n.IsByFork:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		if err := encoder.Encode(n.ByForkBlockNumber); err != nil {
			return err
		}

		return encoder.Encode(n.ByForkBlockHash)
	case n.IsPolkadot:
		return encoder.PushByte(2)
	case n.IsKusama:
		return encoder.PushByte(3)
	case n.IsWestend:
		return encoder.PushByte(4)
	case n.IsRococo:
		return encoder.PushByte(5)
	case n.IsWococo:
		return encoder.PushByte(6)
	case n.IsEthereum:
		if err := encoder.PushByte(7); err != nil {
			return err
		}
		return encoder.Encode(n.Ethereum)
	case n.IsBitcoinCore:
		return encoder.PushByte(8)
	case n.IsBitcoinCash:
		return encoder.PushByte(9)
	}

	return nil
}
