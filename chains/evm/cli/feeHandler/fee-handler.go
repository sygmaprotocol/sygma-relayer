package feeHandler

import (
	"fmt"
	"github.com/ChainSafe/chainbridge-core/chains/evm/cli/flags"
	"github.com/spf13/cobra"
)

var FeeHandlerCmd = &cobra.Command{
	Use:   "fee-handler",
	Short: "Set of commands for interacting with an feeHandler contract",
	Long:  "Set of commands for interacting with an feeHandler contract",
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
	FeeHandlerCmd.AddCommand(setFeeOracleCmd)
	FeeHandlerCmd.AddCommand(setFeePropertiesCmd)
	FeeHandlerCmd.AddCommand(changeFeeCmd)
	FeeHandlerCmd.AddCommand(distributeFeeCmd)
}
