package utils

import "github.com/spf13/cobra"

var UtilsCLI = &cobra.Command{
	Use:   "utils",
	Short: "utility commands that help with random stuff",
}

func init() {
	UtilsCLI.AddCommand(derivateSS58AccountFromPKCMD)
}
