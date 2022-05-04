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

var changeFeeCmd = &cobra.Command{
	Use:   "change-fee",
	Short: "Change the fee in basic fee handler",
	Long:  "The change-fee subcommand sets the new fee to the basic fee handler. Only call this command if basicFeeHandler is deployed",
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
		return ChangeFeeCmd(cmd, args, feeHandler.NewBasicFeeHandlerContract(c, BasicFeeHandlerAddr, t))
	},
	Args: func(cmd *cobra.Command, args []string) error {
		err := ValidateChangeFeeFlags(cmd, args)
		if err != nil {
			return err
		}

		err = ProcessChangeFeeFlags(cmd, args)
		return err
	},
}

func BindChangeFeeFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&FeeHandler, "fee_handler", "", "Fee handler contract address")
	cmd.Flags().Uint64Var(&Fee, "fee", 0, "Fee to be taken when making a deposit (in ETH, decimals are allowed)")
	flags.MarkFlagsAsRequired(cmd, "fee_handler", "fee")
}

func init() {
	BindChangeFeeFlags(changeFeeCmd)
}

func ValidateChangeFeeFlags(cmd *cobra.Command, args []string) error {
	if !common.IsHexAddress(FeeHandler) {
		return fmt.Errorf("invalid fee handler address %s", FeeHandler)
	}
	return nil
}

func ProcessChangeFeeFlags(cmd *cobra.Command, args []string) error {
	BasicFeeHandlerAddr = common.HexToAddress(FeeHandler)

	return nil
}

func ChangeFeeCmd(cmd *cobra.Command, args []string, contract *feeHandler.BasicFeeHandlerContract) error {
	tx, err := contract.ChangeFee(Fee, transactor.TransactOptions{GasLimit: gasLimit})
	if err != nil {
		log.Error().Err(fmt.Errorf("failed to set fee oracle address. error: %v", err))
		return err
	}

	log.Info().Msgf("New fee is set up with transaction: %s", tx.Hex())
	return nil
}
