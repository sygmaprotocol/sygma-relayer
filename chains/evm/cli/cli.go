package cli

import (
	coreCLI "github.com/ChainSafe/chainbridge-core/chains/evm/cli"
	"github.com/ChainSafe/chainbridge-core/chains/evm/cli/logger"
	"github.com/ChainSafe/sygma/chains/evm/cli/deploy"
	"github.com/spf13/cobra"
)

// BindCLI is public function to be invoked in example-app's cobra command
func BindCLI(cli *cobra.Command) {
	cli.AddCommand(EVMRootCLI)
}

var EVMRootCLI = &cobra.Command{
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

	coreCLI.BindEVMCLIFlags(EVMRootCLI)

	// add commands to evm-cli root
	// deploy
	EVMRootCLI.AddCommand(deploy.DeployCLI)

}
