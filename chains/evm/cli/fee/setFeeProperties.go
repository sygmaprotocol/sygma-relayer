package fee

import (
	"errors"

	"github.com/ChainSafe/sygma-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/flags"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/initialize"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/logger"
	"github.com/ChainSafe/sygma-core/util"
	"github.com/ChainSafe/sygma/chains/evm/calls/contracts/feeHandler"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var setFeeProperties = &cobra.Command{
	Use:   "setFeeProperties",
	Short: "Set gasUsed and feePercent",
	Long:  "Set feeHandlerWithOracle gasUsed and feePercent",
	PreRun: func(cmd *cobra.Command, args []string) {
		logger.LoggerMetadata(cmd.Name(), cmd.Flags())
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return util.CallPersistentPreRun(cmd, args)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		err := validateFeePropertiesFlags(cmd, args)
		if err != nil {
			return err
		}
		err = processFeePropertiesFlags(cmd, args)
		return err
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
		return SetFeeProperties(cmd, args, feeHandler.NewFeeHandlerWithOracleContract(c, FeeHandlerAddress, t))
	},
}

func validateFeePropertiesFlags(cmd *cobra.Command, args []string) error {
	if !common.IsHexAddress(FeeHandlerAddressStr) {
		return errors.New("invalid fee handler address")
	}
	return nil
}

func processFeePropertiesFlags(cmd *cobra.Command, args []string) error {
	FeeHandlerAddress = common.HexToAddress(FeeHandlerAddressStr)
	return nil
}

func BindSetFeePropertiesFlags(cmd *cobra.Command) {
	cmd.Flags().Uint32Var(&gasUsed, "gasUsed", 100000, "Amount of gas that current resourceID spend on the destination network")
	cmd.Flags().Uint16Var(&feePercent, "feePercent", 0, "Fee percent that should be collected")
	cmd.Flags().StringVar(&FeeHandlerAddressStr, "feeHandler", "", "FeeHandler to set")
	flags.MarkFlagsAsRequired(cmd, "gasUsed", "feeHandler", "feePercent")
}

func init() {
	BindSetFeePropertiesFlags(setFeeProperties)
}

func SetFeeProperties(cmd *cobra.Command, args []string, contract *feeHandler.FeeHandlerWithOracleContract) error {
	h, err := contract.SetFeeProperties(gasUsed, feePercent, transactor.TransactOptions{GasLimit: gasLimit})
	if err != nil {
		return err
	}
	log.Info().Msgf("Properties for FeeHandler %s set gasLimit %v feePercent %v with tx %s", FeeHandlerAddress.Hex(), gasUsed, feePercent, h.Hex())
	return nil
}
