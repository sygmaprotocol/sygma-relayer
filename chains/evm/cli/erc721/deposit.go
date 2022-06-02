package erc721

import (
	"encoding/hex"
	"fmt"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/cli/initialize"
	"github.com/ChainSafe/chainbridge-core/util"
	"math/big"

	"github.com/ChainSafe/chainbridge-core/chains/evm/cli/flags"
	"github.com/ChainSafe/chainbridge-core/chains/evm/cli/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var depositCmd = &cobra.Command{
	Use:   "deposit",
	Short: "Deposit an ERC721 token",
	Long:  "The deposit subcommand creates a new ERC721 token deposit on the bridge contract",
	PreRun: func(cmd *cobra.Command, args []string) {
		logger.LoggerMetadata(cmd.Name(), cmd.Flags())
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return util.CallPersistentPreRun(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := initialize.InitializeClient(url, senderKeyPair)
		if err != nil {
			return err
		}
		t, err := initialize.InitializeTransactor(gasPrice, evmtransaction.NewTransaction, c, prepare)
		if err != nil {
			return err
		}
		return DepositCmd(cmd, args, bridge.NewBridgeContract(c, BridgeAddr, t))
	},
	Args: func(cmd *cobra.Command, args []string) error {
		err := ValidateDepositFlags(cmd, args)
		if err != nil {
			return err
		}

		err = ProcessDepositFlags(cmd, args)
		return err
	},
}

func BindDepositFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&Recipient, "recipient", "", "Recipient address")
	cmd.Flags().StringVar(&Bridge, "bridge", "", "Bridge contract address")
	cmd.Flags().Uint8Var(&FromDomainID, "from-domain", 0, "Source domain ID(required when fee handler with oracle is in use)")
	cmd.Flags().Uint8Var(&ToDomainID, "to-domain", 0, "Destination domain ID")
	cmd.Flags().StringVar(&ResourceID, "resource", "", "Resource ID for transfer")
	cmd.Flags().StringVar(&Token, "token", "", "ERC721 token ID")
	cmd.Flags().StringVar(&Metadata, "metadata", "", "ERC721 token metadata")
	cmd.Flags().StringVar(&Priority, "priority", "none", "Transaction priority speed (default: medium)")
	cmd.Flags().Uint64Var(&DestNativeTokenDecimals, "dest-native-token-decimals", 0, "Destination domain native token decimals(required when fee handler with oracle is in use)")
	cmd.Flags().Uint64Var(&DestGasPrice, "dest-gas-price", 0, "Destination domain gas price(required when fee handler with oracle is in use)")
	cmd.Flags().StringVar(&BaseRate, "ber", "", "Base rate(required when fee handler with oracle is in use)")
	cmd.Flags().Int64Var(&ExpirationTimestamp, "expire-timestamp", 0, "Rate expire timestamp in unix time, the number of seconds elapsed since January 1, 1970 UTC(required when fee handler with oracle is in use)")
	cmd.Flags().StringVar(&FeeOracleSignature, "fee-oracle-signature", "", "Signature of the fee oracle in hex string without prefix(required when fee handler with oracle is in use)")
	cmd.Flags().BoolVar(&FeeHandlerWithOracle, "fee-handler-with-oracle", false, "Indicator if fee handler with oracle is in use")
	flags.MarkFlagsAsRequired(cmd, "recipient", "bridge", "to-domain", "resource", "token")
}

func init() {
	BindDepositFlags(depositCmd)
}

func ValidateDepositFlags(cmd *cobra.Command, args []string) error {
	if !common.IsHexAddress(Recipient) {
		return fmt.Errorf("invalid recipient address")
	}
	if !common.IsHexAddress(Bridge) {
		return fmt.Errorf("invalid bridge address")
	}
	switch Priority {
	case "none", "slow", "medium", "fast":
		return nil
	default:
		return fmt.Errorf("invalid priority value %s, supported priorities: \"slow|medium|fast\"", Priority)
	}
}

func ProcessDepositFlags(cmd *cobra.Command, args []string) error {
	RecipientAddr = common.HexToAddress(Recipient)
	BridgeAddr = common.HexToAddress(Bridge)

	var ok bool
	TokenId, ok = big.NewInt(0).SetString(Token, 10)
	if !ok {
		return fmt.Errorf("invalid token id value")
	}

	ResourceId, err = flags.ProcessResourceID(ResourceID)
	if FeeOracleSignature != "" {
		ValidFeeOracleSignature, err = hex.DecodeString(FeeOracleSignature)
	}
	return err
}

func DepositCmd(cmd *cobra.Command, args []string, bridgeContract *bridge.BridgeContract) error {
	txHash, err := bridgeContract.Erc721Deposit(
		TokenId, Metadata, RecipientAddr, ResourceId,
		BaseRate, BaseRate, big.NewInt(int64(DestGasPrice)),
		ExpirationTimestamp, FromDomainID, ToDomainID, int64(DestNativeTokenDecimals), int64(DestNativeTokenDecimals),
		ValidFeeOracleSignature, FeeHandlerWithOracle,
		transactor.TransactOptions{GasLimit: gasLimit, Priority: transactor.TxPriorities[Priority]},
	)
	if err != nil {
		return err
	}

	log.Info().Msgf(
		`erc721 deposit hash: %s
		%s token were transferred to %s from %s`,
		txHash.Hex(),
		TokenId.String(),
		RecipientAddr.Hex(),
		senderKeyPair.CommonAddress().String(),
	)
	return nil
}
