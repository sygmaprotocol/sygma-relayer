package cli

import (
	"github.com/ChainSafe/sygma-relayer/chains/evm/cli"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ChainSafe/chainbridge-core/flags"
)

var (
	rootCMD = &cobra.Command{
		Use: "",
	}
)

func init() {
	flags.BindFlags(rootCMD)
	rootCMD.PersistentFlags().String("name", "", "relayer name")
	_ = viper.BindPFlag("name", rootCMD.PersistentFlags().Lookup("name"))
}

func Execute() {
	rootCMD.AddCommand(runCMD, peerInfoCMD, cli.EVMRootCLI)
	if err := rootCMD.Execute(); err != nil {
		log.Fatal().Err(err).Msg("failed to execute root cmd")
	}
}
