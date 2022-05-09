package feeHandler

import (
	"github.com/ChainSafe/chainbridge-core/types"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/crypto/secp256k1"

	"github.com/ethereum/go-ethereum/common"
)

// flag vars
var (
	FeeHandler           string
	FeeOracleAddress     string
	GasUsed              uint32
	FeePercent           uint16
	Fee                  uint64
	DistributionArray    []string
	Decimals             uint64
	ResourceID           string
	FeeHandlerWithOracle bool
)

// processed flag vars
var (
	FeeHandlerWithOracleAddr common.Address
	BasicFeeHandlerAddr      common.Address
	FeeOracleAddr            common.Address
	ResourceIdBytesArr       types.ResourceID
	DistributionAddressArray []common.Address
	DistributionAmountArray  []*big.Int
)

// global flags
var (
	url           string
	gasLimit      uint64
	gasPrice      *big.Int
	senderKeyPair *secp256k1.Keypair
	prepare       bool
)
