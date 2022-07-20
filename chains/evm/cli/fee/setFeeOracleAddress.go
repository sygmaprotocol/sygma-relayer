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

var setFeeOracleAddress = &cobra.Command{
	Use:   "setFeeOracle",
	Short: "Set fee oracle address on FeeHandler",
	Long:  "Fee oracle uses private key to sign rates data, this function sets address that corresponds to FeeOracle private key",
	PreRun: func(cmd *cobra.Command, args []string) {
		logger.LoggerMetadata(cmd.Name(), cmd.Flags())
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return util.CallPersistentPreRun(cmd, args)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		err := validateFlags(cmd, args)
		if err != nil {
			return err
		}
		err = processFlags(cmd, args)
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
		return SetFeeOracle(cmd, args, feeHandler.NewFeeHandlerWithOracleContract(c, FeeHandlerAddress, t))
	},
}

func validateFlags(cmd *cobra.Command, args []string) error {
	if !common.IsHexAddress(FeeOracleAddressStr) {
		return errors.New("invalid feeoracle address")
	}
	if !common.IsHexAddress(FeeHandlerAddressStr) {
		return errors.New("invalid fee handler address")
	}
	return nil
}

func processFlags(cmd *cobra.Command, args []string) error {
	FeeOracleAddress = common.HexToAddress(FeeOracleAddressStr)
	FeeHandlerAddress = common.HexToAddress(FeeHandlerAddressStr)
	return nil
}

func BindSetFeeOracleFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&FeeOracleAddressStr, "feeOracle", "", "Fee oracle address")
	cmd.Flags().StringVar(&FeeHandlerAddressStr, "feeHandler", "", "FeeHandler to set")
	flags.MarkFlagsAsRequired(cmd, "feeOracle", "feeHandler")
}

func init() {
	BindSetFeeOracleFlags(setFeeOracleAddress)
}

func SetFeeOracle(cmd *cobra.Command, args []string, contract *feeHandler.FeeHandlerWithOracleContract) error {
	h, err := contract.SetFeeOracle(FeeOracleAddress, transactor.TransactOptions{GasLimit: gasLimit})
	if err != nil {
		return err
	}
	log.Info().Msgf("FeeOracle address %s set for FeeHandler %s by TX: %s", FeeOracleAddress.Hex(), contract.ContractAddress().Hex(), h.Hex())
	return nil
}
