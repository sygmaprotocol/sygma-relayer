package deploy

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ChainSafe/sygma-core/crypto/secp256k1"
)

var (
	// Flags for all EVM Deploy CLI commands
	Bridge               bool
	BridgeAddress        string
	DeployAll            bool
	DomainId             uint8
	GenericHandler       bool
	Erc20                bool
	Erc20Handler         bool
	Erc20Name            string
	Erc20Symbol          string
	Erc721               bool
	Erc721Handler        bool
	Erc721Name           string
	Erc721Symbol         string
	Erc721BaseURI        string
	FeeHandlerWithOracle bool
	RelayerThreshold     uint64
	Relayers             []string
	Admins               []string
	AdminFunctions       []string
	FeeOracleAddressStr  string
	ResourceID           string
	FeePercent           uint16
	FeeGasUsed           uint32
)

var (
	FeeOracleAddress common.Address
)

// global flags
var (
	senderKeyPair *secp256k1.Keypair
	url           string
)
