package feeHandler

import (
	"math/big"

	"github.com/ChainSafe/chainbridge-core/crypto/secp256k1"

	"github.com/ethereum/go-ethereum/common"
)

// flag vars
var (
	FeeOracleAddress string
	GasUsed          uint32
	FeePercent       uint16
)

// processed flag vars
var (
	FeeHandlerWithOracleAddr common.Address
	FeeOracleAddr            common.Address
)

// global flags
var (
	url           string
	gasLimit      uint64
	gasPrice      *big.Int
	senderKeyPair *secp256k1.Keypair
	prepare       bool
)
