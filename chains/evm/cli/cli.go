package cli

import (
	coreCLI "github.com/ChainSafe/sygma-core/chains/evm/cli"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/account"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/admin"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/bridge"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/erc20"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/erc721"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/logger"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/utils"
	"github.com/ChainSafe/sygma/chains/evm/cli/deploy"
	"github.com/ChainSafe/sygma/chains/evm/cli/fee"
	"github.com/ChainSafe/sygma/chains/evm/cli/local"
	"github.com/spf13/cobra"
)

// BindCLI is public function to be invoked in example-app's cobra command
func BindCLI(cli *cobra.Command) {
	cli.AddCommand(HubRootCLI)
}

var HubRootCLI = &cobra.Command{
	Use:   "evm-cli",
	Short: "EVM CLI",
	Long:  "Root command for starting EVM CLI",
	PreRun: func(cmd *cobra.Command, args []string) {
		logger.LoggerMetadata(cmd.Name(), cmd.Flags())
	},
	// empty Run function to enable cobra PreRun - without this PreRun is never executed
	Run: func(cmd *cobra.Command, args []string) {},
}

func init() {
	// persistent flags
	// to be used across all evm-cli commands (i.e. global)

	coreCLI.BindEVMCLIFlags(HubRootCLI)

	// add commands to evm-cli root

	// admin
	HubRootCLI.AddCommand(admin.AdminCmd)

	// bridge
	HubRootCLI.AddCommand(bridge.BridgeCmd)

	// erc20
	HubRootCLI.AddCommand(erc20.ERC20Cmd)

	// erc721
	HubRootCLI.AddCommand(erc721.ERC721Cmd)

	// account
	HubRootCLI.AddCommand(account.AccountRootCMD)

	// utils
	HubRootCLI.AddCommand(utils.UtilsCmd)

	// add commands to evm-cli root
	// deploy
	HubRootCLI.AddCommand(deploy.DeployEVM)

	// add commands to evm-cli root
	// local setup
	HubRootCLI.AddCommand(local.LocalSetupCmd)

	HubRootCLI.AddCommand(deploy.DeplotTestnetCMD)

	HubRootCLI.AddCommand(fee.FeeHandlerCmd)

}
