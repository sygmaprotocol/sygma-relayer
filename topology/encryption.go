// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package topology

import (
	"crypto/aes"
	"crypto/cipher"
)

type AESEncryption struct {
	cipher cipher.Block
}

func NewAESEncryption(key []byte) (*AESEncryption, error) {
	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return &AESEncryption{
		cipher: cipher,
	}, nil
}

func (ae *AESEncryption) Encrypt(data []byte) []byte {
	dst := make([]byte, len(data))
	ae.cipher.Encrypt(dst, data)
	return dst
}

func (ae *AESEncryption) Decrypt(data []byte) []byte {
	dst := make([]byte, len(data))
	ae.cipher.Decrypt(dst, data)
	return dst
}
