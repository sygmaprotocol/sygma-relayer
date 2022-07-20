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

var setFeeHandler = &cobra.Command{
	Use:   "setFeeHandler",
	Short: "Set FeeHandler on FeeHandlerRouter",
	Long:  "Set fee handler for designated resourceID and destiantion domainID",
	PreRun: func(cmd *cobra.Command, args []string) {
		logger.LoggerMetadata(cmd.Name(), cmd.Flags())
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return util.CallPersistentPreRun(cmd, args)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		err := ValidateSetFeeHandlerFlags(cmd, args)
		if err != nil {
			return err
		}

		err = ProcessSetFeeHandlerFlags(cmd, args)
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
		return SetFeeHandlerCMD(cmd, args, feeHandler.NewFeeRouter(c, FeeRouterAddress, t))
	},
}

func ValidateSetFeeHandlerFlags(cmd *cobra.Command, args []string) error {
	if !common.IsHexAddress(FeeRouterAddressStr) {
		return errors.New("invalid feeRouter address")
	}
	if !common.IsHexAddress(FeeHandlerAddressStr) {
		return errors.New("invalid feeHandler address")
	}
	return nil
}

func ProcessSetFeeHandlerFlags(cmd *cobra.Command, args []string) error {
	var err error
	FeeRouterAddress = common.HexToAddress(FeeRouterAddressStr)
	FeeHandlerAddress = common.HexToAddress(FeeHandlerAddressStr)
	ResourceIDBytesArr, err = flags.ProcessResourceID(ResourceID)
	return err
}

func BindSetFeeHandlerFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&FeeRouterAddressStr, "feeRouter", "", "Fee router")
	cmd.Flags().StringVar(&FeeHandlerAddressStr, "feeHandler", "", "FeeHandler to set")
	cmd.Flags().Uint8Var(&DestDomainID, "domainID", 0, "FeeHandler will be used for this destDomainID")
	cmd.Flags().StringVar(&ResourceID, "resourceID", "", "FeeHander will be used for this resourceID")
	flags.MarkFlagsAsRequired(cmd, "feeRouter", "feeHandler", "domainID", "resourceID")
}

func init() {
	BindSetFeeHandlerFlags(setFeeHandler)
}

func SetFeeHandlerCMD(cmd *cobra.Command, args []string, contract *feeHandler.FeeRouter) error {
	log.Debug().Msgf(`
Setting FeeHandler
FeeRouter address: %s
FeeHandler address: %s
DestDomainID: %v
resourceID: %x`,
		FeeRouterAddressStr, FeeHandlerAddressStr, DestDomainID, ResourceIDBytesArr)

	h, err := contract.AdminSetResourceHandler(DestDomainID, ResourceIDBytesArr, FeeHandlerAddress, transactor.TransactOptions{GasLimit: gasLimit})
	if err != nil {
		return err
	}
	log.Info().Msgf("FeeHandler %s has set for domain %v and resourceID %x with hash %s", FeeHandlerAddressStr, DestDomainID, ResourceIDBytesArr, h.Hex())
	return nil
}
