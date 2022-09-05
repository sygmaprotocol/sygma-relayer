package cli

import (
	"github.com/ChainSafe/sygma-relayer/app"
	"github.com/spf13/cobra"
)

var (
	runCMD = &cobra.Command{
		Use:   "run",
		Short: "Run app",
		Long:  "Run app",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Run(); err != nil {
				return err
			}
			return nil
		},
	}
)
