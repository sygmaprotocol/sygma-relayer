package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ChainSafe/sygma-core/chains/evm/cli/logger"

	"github.com/ChainSafe/chainbridge-hub/chains/evm/cli/deploy"
	"github.com/ChainSafe/chainbridge-hub/chains/evm/cli/local"
)

// BindCLI is public function to be invoked in example-app's cobra command
func BindCLI(cli *cobra.Command) {
	cli.AddCommand(EvmRootCLI)
}

var EvmRootCLI = &cobra.Command{
	Use:   "evm-cli",
	Short: "EVM CLI",
	Long:  "Root command for starting EVM CLI",
	PreRun: func(cmd *cobra.Command, args []string) {
		logger.LoggerMetadata(cmd.Name(), cmd.Flags())
	},
	// empty Run function to enable cobra PreRun - without this PreRun is never executed
	Run: func(cmd *cobra.Command, args []string) {},
}

var (
	// Flags for all EVM CLI commands
	UrlFlagName                = "url"
	GasLimitFlagName           = "gas-limit"
	GasPriceFlagName           = "gas-price"
	NetworkIdFlagName          = "network"
	PrivateKeyFlagName         = "private-key"
	JsonWalletFlagName         = "json-wallet"
	JsonWalletPasswordFlagName = "json-wallet-password"
	Prepare                    = "prepare"
)

func BindEVMCLIFlags(evmRootCLI *cobra.Command) {
	evmRootCLI.PersistentFlags().String(UrlFlagName, "ws://localhost:8545", "URL of the node to receive RPC calls")
	evmRootCLI.PersistentFlags().Uint64(GasLimitFlagName, 6721975, "Gas limit to be used in transactions")
	evmRootCLI.PersistentFlags().Uint64(GasPriceFlagName, 0, "Used as upperLimitGasPrice for transactions if not 0. Transactions gasPrice is defined by estimating it on network for pre London fork networks and by estimating BaseFee and MaxTipFeePerGas in post London networks")
	evmRootCLI.PersistentFlags().Uint64(NetworkIdFlagName, 0, "ID of the Network")
	evmRootCLI.PersistentFlags().String(PrivateKeyFlagName, "", "Private key to use")
	evmRootCLI.PersistentFlags().String(JsonWalletFlagName, "", "Encrypted JSON wallet")
	evmRootCLI.PersistentFlags().String(JsonWalletPasswordFlagName, "", "Password for encrypted JSON wallet")
	evmRootCLI.PersistentFlags().Bool(Prepare, false, "Generate calldata for command")

	_ = viper.BindPFlag(UrlFlagName, evmRootCLI.PersistentFlags().Lookup(UrlFlagName))
	_ = viper.BindPFlag(GasLimitFlagName, evmRootCLI.PersistentFlags().Lookup(GasLimitFlagName))
	_ = viper.BindPFlag(GasPriceFlagName, evmRootCLI.PersistentFlags().Lookup(GasPriceFlagName))
	_ = viper.BindPFlag(NetworkIdFlagName, evmRootCLI.PersistentFlags().Lookup(NetworkIdFlagName))
	_ = viper.BindPFlag(PrivateKeyFlagName, evmRootCLI.PersistentFlags().Lookup(PrivateKeyFlagName))
	_ = viper.BindPFlag(JsonWalletFlagName, evmRootCLI.PersistentFlags().Lookup(JsonWalletFlagName))
	_ = viper.BindPFlag(JsonWalletPasswordFlagName, evmRootCLI.PersistentFlags().Lookup(JsonWalletPasswordFlagName))
	_ = viper.BindPFlag(Prepare, evmRootCLI.PersistentFlags().Lookup(Prepare))
}

func init() {
	// persistent flags
	// to be used across all evm-cli commands (i.e. global)
	BindEVMCLIFlags(EvmRootCLI)

	// add commands to evm-cli root
	// deploy
	EvmRootCLI.AddCommand(deploy.DeployEVM)

	// add commands to evm-cli root
	// local setup
	EvmRootCLI.AddCommand(local.LocalSetupCmd)

}
