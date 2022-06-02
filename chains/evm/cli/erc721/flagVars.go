package erc721

import (
	"math/big"

	"github.com/ChainSafe/chainbridge-core/crypto/secp256k1"
	"github.com/ChainSafe/chainbridge-core/types"
	"github.com/ethereum/go-ethereum/common"
)

// flag vars
var (
	Erc721Address           string
	Dst                     string
	Token                   string
	Metadata                string
	Recipient               string
	Bridge                  string
	FromDomainID            uint8
	ToDomainID              uint8
	ResourceID              string
	Minter                  string
	Priority                string
	DestNativeTokenDecimals uint64
	DestGasPrice            uint64
	BaseRate                string
	ExpirationTimestamp     int64
	FeeOracleSignature      string
	FeeHandlerWithOracle    bool
)

// processed flag vars
var (
	Erc721Addr              common.Address
	DstAddress              common.Address
	TokenId                 *big.Int
	RecipientAddr           common.Address
	BridgeAddr              common.Address
	ResourceId              types.ResourceID
	MinterAddr              common.Address
	ValidFeeOracleSignature []byte
)

// global flags
var (
	url           string
	gasLimit      uint64
	gasPrice      *big.Int
	senderKeyPair *secp256k1.Keypair
	prepare       bool
	err           error
)
