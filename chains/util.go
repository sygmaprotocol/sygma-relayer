package chains

import (
	"math/big"
)

// CalculateStartingBlock returns first block number (smaller or equal) that is dividable with block confirmations
func CalculateStartingBlock(startBlock *big.Int, blockConfirmations *big.Int) *big.Int {
	mod := big.NewInt(0).Mod(startBlock, blockConfirmations)
	startBlock.Sub(startBlock, mod)
	return startBlock
}
