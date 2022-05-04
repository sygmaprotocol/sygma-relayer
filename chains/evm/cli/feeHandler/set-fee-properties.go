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

var setFeePropertiesCmd = &cobra.Command{
	Use:   "set-fee-properties",
	Short: "Set the fee properties to fee handler",
	Long:  "The set-fee-properties subcommand sets the fee handler properties to fee handler. Only call this command if feeHandlerWithOracle is deployed",
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
		return SetFeePropertiesCmd(cmd, args, feeHandler.NewFeeHandlerWithOracleContract(c, FeeHandlerWithOracleAddr, t))
	},
	Args: func(cmd *cobra.Command, args []string) error {
		err := ValidateSetFeePropertiesFlags(cmd, args)
		if err != nil {
			return err
		}

		err = ProcessSetFeePropertiesFlags(cmd, args)
		return err
	},
}

func BindSetFeePropertiesFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&FeeHandler, "fee_handler", "", "Fee handler contract address")
	cmd.Flags().Uint32Var(&GasUsed, "gas_used", 0, "Gas used for transfer")
	cmd.Flags().Uint16Var(&FeePercent, "fee_percent", 0, "Additional amount added to fee amount. total fee = fee + fee * fee_percent")
	flags.MarkFlagsAsRequired(cmd, "fee_handler", "gas_used", "fee_percent")
}

func init() {
	BindSetFeePropertiesFlags(setFeePropertiesCmd)
}

func ValidateSetFeePropertiesFlags(cmd *cobra.Command, args []string) error {
	if !common.IsHexAddress(FeeHandler) {
		return fmt.Errorf("invalid fee handler address %s", FeeHandler)
	}
	if GasUsed == 0 {
		return fmt.Errorf("invalid gas_used value: %v", GasUsed)
	}
	return nil
}

func ProcessSetFeePropertiesFlags(cmd *cobra.Command, args []string) error {
	FeeHandlerWithOracleAddr = common.HexToAddress(FeeHandler)

	return nil
}

func SetFeePropertiesCmd(cmd *cobra.Command, args []string, contract *feeHandler.FeeHandlerWithOracleContract) error {
	tx, err := contract.SetFeeProperties(GasUsed, FeePercent, transactor.TransactOptions{GasLimit: gasLimit})
	if err != nil {
		log.Error().Err(fmt.Errorf("failed to set fee properties. error: %v", err))
		return err
	}

	log.Info().Msgf("Fee oracle properties setup with transaction: %s", tx.Hex())
	return nil
}
