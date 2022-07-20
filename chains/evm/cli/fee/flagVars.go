package fee

import (
	"math/big"

	"github.com/ChainSafe/sygma-core/crypto/secp256k1"
	"github.com/ChainSafe/sygma-core/types"
	"github.com/ethereum/go-ethereum/common"
)

//flag vars
var (
	FeeRouterAddressStr  string
	FeeHandlerAddressStr string
	FeeOracleAddressStr  string
	DestDomainID         uint8
	ResourceID           string
	gasUsed              uint32
	feePercent           uint16
)

//processed flag vars
var (
	FeeRouterAddress   common.Address
	FeeOracleAddress   common.Address
	FeeHandlerAddress  common.Address
	ResourceIDBytesArr types.ResourceID
)

// global flags
var (
	senderKeyPair *secp256k1.Keypair
	url           string
	gasPrice      *big.Int
	prepare       bool
	gasLimit      uint64
)
