package utils

import "github.com/spf13/cobra"

var UtilsCLI = &cobra.Command{
	Use:   "utils",
	Short: "set of utility CLI commands",
}

func init() {
	UtilsCLI.AddCommand(derivateSS58AccountFromPKCMD)
}
