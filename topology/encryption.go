// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package topology

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
)

type AESEncryption struct {
	block cipher.Block
}

func NewAESEncryption(key []byte) (*AESEncryption, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return &AESEncryption{
		block: block,
	}, nil
}

func (ae *AESEncryption) Decrypt(data string) []byte {
	iv := data[:aes.BlockSize]
	bytes, _ := hex.DecodeString(data[aes.BlockSize:])

	stream := cipher.NewCTR(ae.block, []byte(iv))
	dst := make([]byte, len(bytes))
	stream.XORKeyStream(dst, bytes)
	return dst
}
