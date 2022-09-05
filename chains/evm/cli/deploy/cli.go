package deploy

import (
	"github.com/ChainSafe/chainbridge-core/chains/evm/cli/logger"
	"github.com/spf13/cobra"
)

var DeployCLI = &cobra.Command{
	Use:   "deploy",
	Short: "deploy related commands",
	Long:  "todo",
	PreRun: func(cmd *cobra.Command, args []string) {
		logger.LoggerMetadata(cmd.Name(), cmd.Flags())
	},
	// empty Run function to enable cobra PreRun - without this PreRun is never executed
	Run: func(cmd *cobra.Command, args []string) {},
}

func init() {
	DeployCLI.AddCommand(deployCustomCLI)
	DeployCLI.AddCommand(deplotTestnet)
	DeployCLI.AddCommand(LocalSetupCmd)
}
