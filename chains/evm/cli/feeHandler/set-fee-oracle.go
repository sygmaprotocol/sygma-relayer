package feeHandler

import (
	"fmt"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/feeHandler"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/cli/flags"
	"github.com/ChainSafe/chainbridge-core/chains/evm/cli/initialize"
	"github.com/ChainSafe/chainbridge-core/chains/evm/cli/logger"
	"github.com/ChainSafe/chainbridge-core/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var setFeeOracleCmd = &cobra.Command{
	Use:   "set-fee-oracle",
	Short: "Set the fee oracle address to fee handler",
	Long:  "The set-fee-oracle subcommand sets the fee oracle address to fee handler. Only call this command if feeHandlerWithOracle is deployed",
	PreRun: func(cmd *cobra.Command, args []string) {
		logger.LoggerMetadata(cmd.Name(), cmd.Flags())
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return util.CallPersistentPreRun(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := initialize.InitializeClient(url, senderKeyPair)
		if err != nil {
			return err
		}
		t, err := initialize.InitializeTransactor(gasPrice, evmtransaction.NewTransaction, c, prepare)
		if err != nil {
			return err
		}
		return SetFeeOracleCmd(cmd, args, feeHandler.NewFeeHandlerWithOracleContract(c, FeeHandlerWithOracleAddr, t))
	},
	Args: func(cmd *cobra.Command, args []string) error {
		err := ValidateSetFeeOracleFlags(cmd, args)
		if err != nil {
			return err
		}

		ProcessSetFeeOracleFlags(cmd, args)
		return nil
	},
}

func BindSetFeeOracleFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&FeeOracleAddress, "fee_oracle", "", "Fee oracle identity address")
	flags.MarkFlagsAsRequired(cmd, "fee_oracle")
}

func init() {
	BindSetFeeOracleFlags(setFeeOracleCmd)
}

func ValidateSetFeeOracleFlags(cmd *cobra.Command, args []string) error {
	if !common.IsHexAddress(FeeOracleAddress) {
		return fmt.Errorf("invalid fee oracle address %s", FeeOracleAddress)
	}
	return nil
}

func ProcessSetFeeOracleFlags(cmd *cobra.Command, args []string) {
	FeeOracleAddr = common.HexToAddress(FeeOracleAddress)
}

func SetFeeOracleCmd(cmd *cobra.Command, args []string, contract *feeHandler.FeeHandlerWithOracleContract) error {
	tx, err := contract.SetFeeOracle(FeeOracleAddr, transactor.TransactOptions{GasLimit: gasLimit})
	if err != nil {
		log.Error().Err(fmt.Errorf("failed to set fee oracle address. error: %v", err))
		return err
	}

	log.Info().Msgf("Fee oracle address setup with transaction: %s", tx.Hex())
	return nil
}
