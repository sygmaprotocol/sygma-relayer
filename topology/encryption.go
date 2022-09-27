// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package topology

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
)

var IV = []byte("1234567812345678")

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

func (ae *AESEncryption) Encrypt(data []byte) string {
	dst := make([]byte, len(data))
	stream := cipher.NewCTR(ae.block, IV)
	stream.XORKeyStream(dst, data)
	return hex.EncodeToString(dst)
}

func (ae *AESEncryption) Decrypt(data string) []byte {
	stream := cipher.NewCTR(ae.block, IV)
	bytes, _ := hex.DecodeString(data)
	dst := make([]byte, len(bytes))
	stream.XORKeyStream(dst, bytes)
	return dst
}
