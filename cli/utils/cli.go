package utils

import "github.com/spf13/cobra"

var UtilsCLI = &cobra.Command{
	Use:   "utils",
	Short: "utility commands that helps with random staff",
}

func init() {
	UtilsCLI.AddCommand(derivateSS58AccountFromPKCMD)
}
