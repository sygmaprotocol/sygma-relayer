// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package util

func SliceTo4Bytes(in []byte) [4]byte {
	var res [4]byte
	copy(res[:], in)
	return res
}
