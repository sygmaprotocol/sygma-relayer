package app

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"

	"github.com/ChainSafe/chainbridge-core/chains/evm/cli/deploy"
	"github.com/ChainSafe/chainbridge-core/flags"
)

var (
	rootCMD = &cobra.Command{
		Use: "",
	}
	runCMD = &cobra.Command{
		Use:   "run",
		Short: "Run app",
		Long:  "Run app",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := Run(); err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	flags.BindFlags(rootCMD)
}

func Execute() {
	log.Info().Msgf("Preparing CLI commands, args %+v", os.Args)
	rootCMD.AddCommand(runCMD, deploy.DeployEVM)
	log.Info().Msg("CLI commands ready for execution")
	if err := rootCMD.Execute(); err != nil {
		log.Fatal().Err(err).Msg("failed to execute root cmd")
	}
}
