package bridge

import (
	"fmt"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/bridge"
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

var adminChangeFeeHandlerCmd = &cobra.Command{
	Use:   "admin-change-fee-handler",
	Short: "Change the fee handler address in bridge by admin",
	Long:  "The admin-change-fee-handler subcommand sets the fee handler address in bridge by admin",
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
		return AdminChangeFeeHandlerCmd(cmd, args, bridge.NewBridgeContract(c, BridgeAddr, t))
	},
	Args: func(cmd *cobra.Command, args []string) error {
		err := ValidateAdminChangeFeeHandlerFlags(cmd, args)
		if err != nil {
			return err
		}

		err = ProcessAdminChangeFeeHandlerFlags(cmd, args)
		return err
	},
}

func BindAdminChangeFeeHandlerFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&Bridge, "bridge", "", "Bridge contract address")
	cmd.Flags().StringVar(&FeeHandler, "fee_handler", "", "Fee handler contract address")
	flags.MarkFlagsAsRequired(cmd, "fee_handler")
}

func init() {
	BindAdminChangeFeeHandlerFlags(adminChangeFeeHandlerCmd)
}

func ValidateAdminChangeFeeHandlerFlags(cmd *cobra.Command, args []string) error {
	if !common.IsHexAddress(Bridge) {
		return fmt.Errorf("invalid bridge address %s", Bridge)
	}
	if !common.IsHexAddress(FeeHandler) {
		return fmt.Errorf("invalid fee handler address %s", FeeHandler)
	}
	return nil
}

func ProcessAdminChangeFeeHandlerFlags(cmd *cobra.Command, args []string) error {
	BridgeAddr = common.HexToAddress(Bridge)
	FeeHandlerAddr = common.HexToAddress(FeeHandler)

	return nil
}

func AdminChangeFeeHandlerCmd(cmd *cobra.Command, args []string, contract *bridge.BridgeContract) error {
	h, err := contract.AdminChangeFeeHandler(
		FeeHandlerAddr, transactor.TransactOptions{GasLimit: gasLimit},
	)
	if err != nil {
		log.Error().Err(err)
		return err
	}

	log.Info().Msgf("Admin changes fee handler with transaction: %s", h.Hex())
	return nil
}
