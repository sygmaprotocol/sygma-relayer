package substrate

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type JunctionV3 struct {
	IsParachain bool
	ParachainID types.UCompact

	IsAccountID32        bool
	AccountID32NetworkID NetworkIDV3
	AccountID            []types.U8

	IsAccountIndex64        bool
	AccountIndex64NetworkID NetworkIDV3
	AccountIndex            types.U64

	IsAccountKey20        bool
	AccountKey20NetworkID NetworkIDV3
	AccountKey            []types.U8

	IsPalletInstance bool
	PalletIndex      types.U8

	IsGeneralIndex bool
	GeneralIndex   types.U128

	IsGeneralKey     bool
	GeneralKeyLength types.U8
	GeneralKey       [32]types.U8

	IsOnlyChild bool

	IsPlurality bool
	BodyID      types.BodyID
	BodyPart    types.BodyPart

	IsGlobalConsensus bool
	GlobalConsensus   NetworkIDV3
}

func (j *JunctionV3) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		j.IsParachain = true

		return decoder.Decode(&j.ParachainID)
	case 1:
		j.IsAccountID32 = true

		if err := decoder.Decode(&j.AccountID32NetworkID); err != nil {
			return nil
		}

		return decoder.Decode(&j.AccountID)
	case 2:
		j.IsAccountIndex64 = true

		if err := decoder.Decode(&j.AccountIndex64NetworkID); err != nil {
			return nil
		}

		return decoder.Decode(&j.AccountIndex)
	case 3:
		j.IsAccountKey20 = true

		if err := decoder.Decode(&j.AccountKey20NetworkID); err != nil {
			return nil
		}

		return decoder.Decode(&j.AccountKey)
	case 4:
		j.IsPalletInstance = true

		return decoder.Decode(&j.PalletIndex)
	case 5:
		j.IsGeneralIndex = true

		return decoder.Decode(&j.GeneralIndex)
	case 6:
		j.IsGeneralKey = true

		if err := decoder.Decode(&j.GeneralKeyLength); err != nil {
			return nil
		}

		return decoder.Decode(&j.GeneralKey)
	case 7:
		j.IsOnlyChild = true
	case 8:
		j.IsPlurality = true

		if err := decoder.Decode(&j.BodyID); err != nil {
			return err
		}

		return decoder.Decode(&j.BodyPart)
	case 9:
		j.IsGlobalConsensus = true
		return decoder.Decode(&j.GlobalConsensus)
	}

	return nil
}

func (j JunctionV3) Encode(encoder scale.Encoder) error { //nolint:funlen
	switch {
	case j.IsParachain:
		if err := encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(j.ParachainID)
	case j.IsAccountID32:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		if err := encoder.Encode(j.AccountID32NetworkID); err != nil {
			return err
		}

		return encoder.Encode(j.AccountID)
	case j.IsAccountIndex64:
		if err := encoder.PushByte(2); err != nil {
			return err
		}

		if err := encoder.Encode(j.AccountIndex64NetworkID); err != nil {
			return err
		}

		return encoder.Encode(j.AccountIndex)
	case j.IsAccountKey20:
		if err := encoder.PushByte(3); err != nil {
			return err
		}

		if err := encoder.Encode(j.AccountKey20NetworkID); err != nil {
			return err
		}

		return encoder.Encode(j.AccountKey)
	case j.IsPalletInstance:
		if err := encoder.PushByte(4); err != nil {
			return err
		}

		return encoder.Encode(j.PalletIndex)
	case j.IsGeneralIndex:
		if err := encoder.PushByte(5); err != nil {
			return err
		}

		return encoder.Encode(j.GeneralIndex)
	case j.IsGeneralKey:
		if err := encoder.PushByte(6); err != nil {
			return err
		}

		if err := encoder.Encode(j.GeneralKeyLength); err != nil {
			return err
		}

		return encoder.Encode(j.GeneralKey)
	case j.IsOnlyChild:
		return encoder.PushByte(7)
	case j.IsPlurality:
		if err := encoder.PushByte(8); err != nil {
			return err
		}

		if err := encoder.Encode(j.BodyID); err != nil {
			return err
		}

		return encoder.Encode(j.BodyPart)

	case j.IsGlobalConsensus:
		if err := encoder.PushByte(9); err != nil {
			return err
		}

		return encoder.Encode(j.GlobalConsensus)
	}

	return nil
}

type JunctionsV3 struct {
	IsHere bool

	IsX2 bool
	X2   [2]JunctionV3

	IsX3 bool
	X3   [3]JunctionV3
}

func (j *JunctionsV3) Decode(decoder scale.Decoder) error { //nolint:dupl
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		j.IsHere = true
	case 2:
		j.IsX2 = true

		return decoder.Decode(&j.X2)
	case 3:
		j.IsX3 = true

		return decoder.Decode(&j.X3)

	}
	return nil
}

func (j JunctionsV3) Encode(encoder scale.Encoder) error {
	switch {
	case j.IsHere:
		return encoder.PushByte(0)
	case j.IsX2:
		if err := encoder.PushByte(2); err != nil {
			return err
		}

		return encoder.Encode(j.X2)
	case j.IsX3:
		if err := encoder.PushByte(3); err != nil {
			return err
		}

		return encoder.Encode(j.X3)
	}

	return nil
}
