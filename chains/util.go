// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package chains

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
)

// CalculateStartingBlock returns first block number (smaller or equal) that is dividable with block confirmations
func CalculateStartingBlock(startBlock *big.Int, blockConfirmations *big.Int) (*big.Int, error) {
	if startBlock == nil || blockConfirmations == nil {
		return nil, fmt.Errorf("startBlock or blockConfirmations can not be nill when calculating CalculateStartingBlock")
	}
	mod := big.NewInt(0).Mod(startBlock, blockConfirmations)
	startBlock.Sub(startBlock, mod)
	return startBlock, nil
}

// CalculateNonce calculates a transfer nonce for networks that don't have smart contracts
func CalculateNonce(blockNumber *big.Int, transactionHash string) uint64 {
	// Convert blockNumber to string
	blockNumberStr := blockNumber.String()

	// Concatenate blockNumberStr and transactionHash with a separator
	concatenatedStr := blockNumberStr + "-" + transactionHash

	// Calculate SHA-256 hash of the concatenated string
	hash := sha256.New()
	hash.Write([]byte(concatenatedStr))
	hashBytes := hash.Sum(nil)

	// XOR fold the hash to get a 64-bit value
	var result uint64
	for i := 0; i < 4; i++ {
		part := binary.BigEndian.Uint64(hashBytes[i*8 : (i+1)*8])
		result ^= part
	}

	return result
}
