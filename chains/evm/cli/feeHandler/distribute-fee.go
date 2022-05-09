package feeHandler

import (
	"fmt"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	callsUtil "github.com/ChainSafe/chainbridge-core/chains/evm/calls"
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
	"math/big"
)

var distributeFeeCmd = &cobra.Command{
	Use:   "distribute-fee",
	Short: "Distribute the fee from fee handler contract to the specified addresses",
	Long:  "The distribute-fee subcommand transfers the fee from fee handler contract to the specified addresses",
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
		return DistributeFeeCmd(cmd, args, c, t)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		err := ValidateDistributeFeeFlags(cmd, args)
		if err != nil {
			return err
		}

		err = ProcessDistributeFeeFlags(cmd, args)
		return err
	},
}

func BindDistributeFeeFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&FeeHandler, "fee-handler", "", "Fee handler contract address")
	cmd.Flags().StringSliceVar(&DistributionArray, "distribution-array", []string{}, "Array of fee distribution, follow the format of \"--distribution-array address,amount,...\" where address,amount must be paired")

	cmd.Flags().StringVar(&ResourceID, "resource-id", "", "ResourceID to be used when making deposits")
	cmd.Flags().Uint64Var(&Decimals, "decimals", 0, "Resource token decimals")
	cmd.Flags().BoolVar(&FeeHandlerWithOracle, "fee-handler-with-oracle", false, "Using Fee handler with oracle. Default is basic fee handler")
	flags.MarkFlagsAsRequired(cmd, "fee-handler", "distribution-array")
}

func init() {
	BindDistributeFeeFlags(distributeFeeCmd)
}

func ValidateDistributeFeeFlags(cmd *cobra.Command, args []string) error {
	if !common.IsHexAddress(FeeHandler) {
		return fmt.Errorf("invalid fee handler address %s", FeeHandler)
	}

	if len(DistributionArray)%2 != 0 {
		return fmt.Errorf("invalid distribution-array length: %d", len(DistributionArray))
	}

	decimals := big.NewInt(18)
	if FeeHandlerWithOracle {
		if len(ResourceID) == 0 || Decimals == 0 {
			return fmt.Errorf("invalid resource-id or resource token decimals: %s, %d. Must provide valid resource-id with decimals if providing fee-handler-with-oracle flag", ResourceID, Decimals)
		}
		decimals = big.NewInt(int64(Decimals))
	}

	DistributionAddressArray = make([]common.Address, 0)
	DistributionAmountArray = make([]*big.Int, 0)
	for i := 0; i < len(DistributionArray); i++ {
		if !common.IsHexAddress(DistributionArray[i]) {
			return fmt.Errorf("invalid address in distribution-array %v", DistributionArray)
		}
		DistributionAddressArray = append(DistributionAddressArray, common.HexToAddress(DistributionArray[i]))

		a, err := callsUtil.UserAmountToWei(DistributionArray[i+1], decimals)
		if err != nil {
			return fmt.Errorf("invalid amount in distribution-array %v", DistributionArray)
		}
		DistributionAmountArray = append(DistributionAmountArray, a)
		i++
	}

	return nil
}

func ProcessDistributeFeeFlags(cmd *cobra.Command, args []string) error {
	var err error

	BasicFeeHandlerAddr = common.HexToAddress(FeeHandler)
	if FeeHandlerWithOracle {
		FeeHandlerWithOracleAddr = common.HexToAddress(FeeHandler)
		ResourceIdBytesArr, err = flags.ProcessResourceID(ResourceID)
	}

	return err
}

func DistributeFeeCmd(cmd *cobra.Command, args []string, c calls.ContractCallerDispatcher, t transactor.Transactor) error {
	var txHash *common.Hash
	var err error
	if FeeHandlerWithOracle {
		fmt.Println("With oracle")
		contract := feeHandler.NewFeeHandlerWithOracleContract(c, FeeHandlerWithOracleAddr, t)
		txHash, err = contract.DistributeFee(ResourceIdBytesArr, DistributionAddressArray, DistributionAmountArray, transactor.TransactOptions{GasLimit: gasLimit})
	} else {
		fmt.Println("basic fee handler")
		contract := feeHandler.NewBasicFeeHandlerContract(c, BasicFeeHandlerAddr, t)
		txHash, err = contract.DistributeFee(DistributionAddressArray, DistributionAmountArray, transactor.TransactOptions{GasLimit: gasLimit})
	}
	if err != nil {
		log.Error().Err(fmt.Errorf("failed to distribute fee. error: %v", err))
		return err
	}

	log.Info().Msgf("Fee has been distributed with transaction: %s", txHash.Hex())
	return nil
}
