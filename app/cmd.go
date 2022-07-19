package app

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ChainSafe/sygma-core/chains/evm/cli/deploy"
	"github.com/ChainSafe/sygma-core/flags"
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
	rootCMD.PersistentFlags().String("name", "", "relayer name")
	_ = viper.BindPFlag("name", rootCMD.PersistentFlags().Lookup("name"))
}

func Execute() {
	rootCMD.AddCommand(runCMD, deploy.DeployEVM)
	if err := rootCMD.Execute(); err != nil {
		log.Fatal().Err(err).Msg("failed to execute root cmd")
	}
}
