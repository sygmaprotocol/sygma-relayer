package fee

import (
	"fmt"

	"github.com/ChainSafe/sygma-core/chains/evm/cli/flags"
	"github.com/spf13/cobra"
)

var FeeHandlerCmd = &cobra.Command{
	Use:   "fee",
	Short: "Set of commands for interacting with a FeeHandlers and Routers",
	Long:  "Set of commands for interacting with a FeeHandlers and Routers",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		// fetch global flag values
		url, gasLimit, gasPrice, senderKeyPair, prepare, err = flags.GlobalFlagValues(cmd)
		if err != nil {
			return fmt.Errorf("could not get global flags: %v", err)
		}
		return nil
	},
}

func init() {
	FeeHandlerCmd.AddCommand(setFeeHandler)
	FeeHandlerCmd.AddCommand(setFeeOracleAddress)
	FeeHandlerCmd.AddCommand(setFeeProperties)
}
